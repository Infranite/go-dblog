package decoder

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	mysqlclient "github.com/go-mysql-org/go-mysql/client"
	gomysql "github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"

	"github.com/Infranite/go-dblog/mysql/common"
	"github.com/Infranite/go-dblog/mysql/decode/events/types"
)

const defaultLiveServerID uint32 = 1001
const defaultLiveReadTimeout = 30 * time.Second

// LiveDecoder streams MySQL replication events through the native binlog parser.
type LiveDecoder struct {
	ctx           context.Context
	source        Source
	syncer        *replication.BinlogSyncer
	streamer      *replication.BinlogStreamer
	decoder       *BinFileDecoder
	startPosition int64
}

type liveConfig struct {
	address  string
	user     string
	password string
	database string
	flavor   string
	serverID uint32
	file     string
	pos      uint32
}

type liveMetadata struct {
	file        string
	pos         uint32
	version     string
	hasChecksum bool
}

// NewLiveDecoder creates a context-aware decoder over a MySQL replication stream.
func NewLiveDecoder(ctx context.Context, source Source, dsn string, opts ...BinFileDecodeOptFunc) (*LiveDecoder, error) {
	cfg, err := parseLiveDSN(dsn)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	option := newBinFileDecodeOption(opts...)
	startPosition := option.StartPos
	option.StartPos = 0
	if cfg.pos == 0 && startPosition > 0 {
		if startPosition > int64(^uint32(0)) {
			return nil, fmt.Errorf("mysql: checkpoint position %d exceeds binlog position range", startPosition)
		}
		cfg.pos = uint32(startPosition)
	}
	meta, err := loadLiveMetadata(cfg)
	if err != nil {
		return nil, err
	}
	// go-mysql negotiates replication streams with @source_binlog_checksum=NONE.
	meta.hasChecksum = false
	if cfg.file == "" {
		cfg.file = meta.file
	}
	if cfg.pos == 0 {
		cfg.pos = meta.pos
	}
	if source.Driver == "" {
		source.Driver = driverMySQL
	}
	source.Name = cfg.file

	rawDecoder := newRawDecoder()
	rawDecoder.Option = option
	rawDecoder.Description = defaultLiveDescription(meta)

	syncer := replication.NewBinlogSyncer(replication.BinlogSyncerConfig{
		ServerID:       cfg.serverID,
		Flavor:         cfg.flavor,
		Host:           hostOnly(cfg.address),
		Port:           portOnly(cfg.address),
		User:           cfg.user,
		Password:       cfg.password,
		RawModeEnabled: true,
		ReadTimeout:    defaultLiveReadTimeout,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	streamer, err := syncer.StartSync(gomysql.Position{Name: cfg.file, Pos: cfg.pos})
	if err != nil {
		syncer.Close()
		return nil, fmt.Errorf("mysql live start sync: %w", err)
	}
	return &LiveDecoder{
		ctx:           ctx,
		source:        source,
		syncer:        syncer,
		streamer:      streamer,
		decoder:       rawDecoder,
		startPosition: startPosition,
	}, nil
}

func (d *LiveDecoder) Events() iter.Seq2[*DblogEvent, error] {
	return func(yield func(*DblogEvent, error) bool) {
		if d == nil || d.streamer == nil || d.decoder == nil {
			return
		}
		for {
			event, err := d.streamer.GetEvent(d.ctx)
			if err != nil {
				if d.ctx.Err() == nil {
					yield(nil, err)
				}
				return
			}
			if event == nil || len(event.RawData) == 0 {
				continue
			}
			native, err := d.decoder.DecodeRawEvent(event.RawData)
			if err != nil {
				yield(nil, err)
				return
			}
			if native == nil || native.Header == nil {
				continue
			}
			if d.startPosition > 0 && native.Header.LogPos <= d.startPosition {
				continue
			}
			if !yield(&DblogEvent{source: d.source, event: native}, nil) {
				return
			}
		}
	}
}

func (d *LiveDecoder) Close() error {
	if d == nil || d.syncer == nil {
		return nil
	}
	d.syncer.Close()
	return nil
}

func parseLiveDSN(dsn string) (liveConfig, error) {
	parsed, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil || parsed.Scheme != "mysql" || parsed.Host == "" {
		return liveConfig{}, fmt.Errorf("mysql: invalid live dsn")
	}
	port, err := parsePort(parsed.Port())
	if err != nil {
		return liveConfig{}, err
	}
	query := parsed.Query()
	serverID, err := parseUint32(query.Get("server_id"), defaultLiveServerID, "server_id")
	if err != nil {
		return liveConfig{}, err
	}
	pos, err := parseUint32(query.Get("pos"), 0, "pos")
	if err != nil {
		return liveConfig{}, err
	}
	password, _ := parsed.User.Password()
	flavor := query.Get("flavor")
	if flavor == "" {
		flavor = "mysql"
	}
	return liveConfig{
		address:  net.JoinHostPort(parsed.Hostname(), strconv.Itoa(int(port))),
		user:     parsed.User.Username(),
		password: password,
		database: strings.TrimPrefix(parsed.Path, "/"),
		flavor:   flavor,
		serverID: serverID,
		file:     firstNonEmpty(query.Get("binlog"), query.Get("file")),
		pos:      pos,
	}, nil
}

func loadLiveMetadata(cfg liveConfig) (liveMetadata, error) {
	conn, err := mysqlclient.Connect(cfg.address, cfg.user, cfg.password, cfg.database)
	if err != nil {
		return liveMetadata{}, fmt.Errorf("mysql live connect: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	meta := liveMetadata{file: cfg.file, pos: cfg.pos, version: "8.0.0"}
	if err := loadServerVersion(conn, &meta); err != nil {
		return liveMetadata{}, err
	}
	if meta.file == "" || meta.pos == 0 {
		if err := loadBinlogPosition(conn, &meta); err != nil {
			return liveMetadata{}, err
		}
	}
	return meta, nil
}

func loadServerVersion(conn *mysqlclient.Conn, meta *liveMetadata) error {
	result, err := conn.Execute("SELECT @@version")
	if err != nil {
		return fmt.Errorf("mysql live read server version: %w", err)
	}
	version, err := result.GetString(0, 0)
	if err != nil {
		return fmt.Errorf("mysql live read server version: %w", err)
	}
	meta.version = version
	return nil
}

func loadBinlogPosition(conn *mysqlclient.Conn, meta *liveMetadata) error {
	result, err := conn.Execute("SHOW BINARY LOG STATUS")
	if err != nil {
		result, err = conn.Execute("SHOW MASTER STATUS")
	}
	if err != nil {
		return fmt.Errorf("mysql live read binlog status: %w", err)
	}
	file, err := result.GetString(0, 0)
	if err != nil {
		return fmt.Errorf("mysql live read binlog file: %w", err)
	}
	pos, err := result.GetUint(0, 1)
	if err != nil {
		return fmt.Errorf("mysql live read binlog position: %w", err)
	}
	meta.file = file
	meta.pos = uint32(pos)
	return nil
}

func defaultLiveDescription(meta liveMetadata) *types.FmtDescEvent {
	return &types.FmtDescEvent{
		BinlogVersion:     4,
		MySQLVersion:      meta.version,
		EventHeaderLength: common.DefaultEventHeaderSize,
		EventTypeHeader:   defaultEventTypeHeader(),
		HasCheckSum:       meta.hasChecksum,
	}
}

func defaultEventTypeHeader() []byte {
	header := make([]byte, common.GTIDTaggedLogEvent)
	header[common.TableMapEvent-1] = 8
	for _, eventType := range []uint8{
		common.WriteRowsEventV0, common.UpdateRowsEventV0, common.DeleteRowsEventV0,
	} {
		header[eventType-1] = 6
	}
	for _, eventType := range []uint8{
		common.WriteRowsEventV1, common.UpdateRowsEventV1, common.DeleteRowsEventV1,
	} {
		header[eventType-1] = 8
	}
	for _, eventType := range []uint8{
		common.WriteRowsEventV2, common.UpdateRowsEventV2, common.DeleteRowsEventV2,
		common.PartialUpdateRowsEvent,
	} {
		header[eventType-1] = 10
	}
	return header
}

func parsePort(port string) (uint16, error) {
	if port == "" {
		return 3306, nil
	}
	value, err := strconv.ParseUint(port, 10, 16)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("mysql: invalid live dsn port %q", port)
	}
	return uint16(value), nil
}

func parseUint32(raw string, fallback uint32, field string) (uint32, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("mysql: invalid live dsn %s %q", field, raw)
	}
	return uint32(value), nil
}

func hostOnly(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return host
}

func portOnly(address string) uint16 {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return 3306
	}
	value, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return 3306
	}
	return uint16(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

var _ interface {
	Events() iter.Seq2[*DblogEvent, error]
	Close() error
} = (*LiveDecoder)(nil)
