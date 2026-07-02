package decoder

import (
	"context"
	"encoding/binary"
	"fmt"
	"iter"
	"net/url"
	"strings"
	"time"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
)

const wireStartLSN = "0/0"

var postgresEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// WireLiveDecoder streams test_decoding output over PostgreSQL logical replication protocol.
type WireLiveDecoder struct {
	ctx           context.Context
	source        dblog.Source
	conn          *pgconn.PgConn
	slot          string
	plugins       []types.EventPlugin
	startPosition int
	position      int
	started       bool
}

// NewWireLiveDecoder creates a decoder over PostgreSQL logical replication protocol.
func NewWireLiveDecoder(
	ctx context.Context,
	source dblog.Source,
	dsn string,
	slot string,
	opts ...Option,
) (*WireLiveDecoder, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, types.ErrReaderRequired
	}
	if strings.TrimSpace(slot) == "" {
		return nil, types.ErrSlotRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}
	config, err := pgconn.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres wire parse dsn: %w", err)
	}
	if config.RuntimeParams == nil {
		config.RuntimeParams = map[string]string{}
	}
	config.RuntimeParams["replication"] = "database"
	conn, err := pgconn.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("postgres wire connect: %w", err)
	}
	if source.Driver == "" {
		source.Driver = types.Driver
	}
	if source.Name == "" {
		source.Name = slot
	}
	cfg := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &WireLiveDecoder{
		ctx:           ctx,
		source:        source,
		conn:          conn,
		slot:          slot,
		plugins:       cfg.eventPlugins,
		startPosition: cfg.startPosition,
	}, nil
}

func (d *WireLiveDecoder) Events() iter.Seq2[dblog.Event, error] {
	return func(yield func(dblog.Event, error) bool) {
		if d == nil || d.conn == nil {
			return
		}
		if !d.started {
			if err := d.startReplication(); err != nil {
				if d.ctx.Err() == nil {
					yield(nil, err)
				}
				return
			}
			d.started = true
		}
		for d.ctx.Err() == nil {
			message, err := d.conn.ReceiveMessage(d.ctx)
			if err != nil {
				if d.ctx.Err() == nil {
					yield(nil, fmt.Errorf("postgres wire receive: %w", err))
				}
				return
			}
			switch message := message.(type) {
			case *pgproto3.CopyData:
				if !d.yieldCopyData(message.Data, yield) {
					return
				}
			case *pgproto3.ErrorResponse:
				yield(nil, pgconn.ErrorResponseToPgError(message))
				return
			}
		}
	}
}

func (d *WireLiveDecoder) startReplication() error {
	d.conn.Frontend().Send(&pgproto3.Query{String: startReplicationQuery(d.slot)})
	if err := d.conn.Frontend().Flush(); err != nil {
		return fmt.Errorf("postgres wire start replication: %w", err)
	}
	for {
		message, err := d.conn.ReceiveMessage(d.ctx)
		if err != nil {
			return fmt.Errorf("postgres wire start replication: %w", err)
		}
		switch message := message.(type) {
		case *pgproto3.CopyBothResponse:
			return nil
		case *pgproto3.ErrorResponse:
			return pgconn.ErrorResponseToPgError(message)
		}
	}
}

func (d *WireLiveDecoder) yieldCopyData(data []byte, yield func(dblog.Event, error) bool) bool {
	change, err := parseWireData(data)
	if err != nil {
		yield(nil, err)
		return false
	}
	if change.lsn > 0 || change.reply {
		if err := d.sendStandbyStatus(change.lsn, change.reply); err != nil {
			yield(nil, err)
			return false
		}
	}
	line := strings.TrimSpace(change.line)
	if line == "" {
		return true
	}
	d.position++
	if d.position <= d.startPosition {
		return true
	}
	event, err := parseLine(d.source, d.position, line, d.plugins)
	if err != nil {
		yield(nil, err)
		return false
	}
	return yield(event, nil)
}

func (d *WireLiveDecoder) sendStandbyStatus(lsn uint64, reply bool) error {
	d.conn.Frontend().Send(&pgproto3.CopyData{Data: standbyStatusData(lsn, time.Now().UTC(), reply)})
	if err := d.conn.Frontend().Flush(); err != nil {
		return fmt.Errorf("postgres wire send standby status: %w", err)
	}
	return nil
}

func (d *WireLiveDecoder) Close() error {
	if d == nil || d.conn == nil {
		return nil
	}
	return d.conn.Close(context.Background())
}

// IsWireReplicationDSN reports whether a PostgreSQL DSN requests replication protocol.
func IsWireReplicationDSN(dsn string) bool {
	dsn = strings.TrimSpace(dsn)
	if parsed, err := url.Parse(dsn); err == nil && parsed.Scheme != "" {
		return strings.EqualFold(parsed.Query().Get("replication"), "database")
	}
	for _, field := range strings.Fields(dsn) {
		key, value, ok := strings.Cut(field, "=")
		if ok && strings.EqualFold(key, "replication") && strings.EqualFold(strings.Trim(value, `'"`), "database") {
			return true
		}
	}
	return false
}

type wireData struct {
	line  string
	lsn   uint64
	reply bool
}

func parseWireData(data []byte) (wireData, error) {
	if len(data) == 0 {
		return wireData{}, fmt.Errorf("postgres wire empty copy data")
	}
	switch data[0] {
	case 'w':
		if len(data) < 25 {
			return wireData{}, fmt.Errorf("postgres wire xlog data too short")
		}
		return wireData{
			lsn:  binary.BigEndian.Uint64(data[9:17]),
			line: string(data[25:]),
		}, nil
	case 'k':
		if len(data) < 18 {
			return wireData{}, fmt.Errorf("postgres wire keepalive too short")
		}
		return wireData{
			lsn:   binary.BigEndian.Uint64(data[1:9]),
			reply: data[17] != 0,
		}, nil
	default:
		return wireData{}, fmt.Errorf("postgres wire unknown copy data type %q", data[0])
	}
}

func standbyStatusData(lsn uint64, now time.Time, reply bool) []byte {
	data := make([]byte, 34)
	data[0] = 'r'
	binary.BigEndian.PutUint64(data[1:9], lsn)
	binary.BigEndian.PutUint64(data[9:17], lsn)
	binary.BigEndian.PutUint64(data[17:25], lsn)
	binary.BigEndian.PutUint64(data[25:33], uint64(now.UTC().Sub(postgresEpoch).Microseconds()))
	if reply {
		data[33] = 1
	}
	return data
}

func startReplicationQuery(slot string) string {
	return fmt.Sprintf(
		"START_REPLICATION SLOT %s LOGICAL %s (\"include-xids\" '1')",
		quoteIdentifier(slot),
		wireStartLSN,
	)
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

var _ dblog.Decoder[dblog.Event] = (*WireLiveDecoder)(nil)
