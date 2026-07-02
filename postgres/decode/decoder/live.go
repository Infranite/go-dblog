package decoder

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
	"github.com/jackc/pgx/v5"
)

const (
	defaultLivePollInterval = 100 * time.Millisecond
	liveChangesQuery        = "SELECT data FROM pg_logical_slot_get_changes($1::name, NULL, NULL, 'include-xids', '1')"
)

type liveQueryer interface {
	Query(context.Context, string, ...any) (liveRows, error)
}

type liveRows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}

type pgxLiveQueryer struct {
	conn *pgx.Conn
}

func (q pgxLiveQueryer) Query(ctx context.Context, query string, args ...any) (liveRows, error) {
	return q.conn.Query(ctx, query, args...)
}

// LiveDecoder streams PostgreSQL logical decoding rows from a live slot.
type LiveDecoder struct {
	ctx           context.Context
	source        dblog.Source
	queryer       liveQueryer
	close         func() error
	slot          string
	plugins       []types.EventPlugin
	startPosition int
	pollInterval  time.Duration
	position      int
}

// NewLiveDecoder creates a decoder over a live PostgreSQL logical decoding slot.
func NewLiveDecoder(
	ctx context.Context,
	source dblog.Source,
	dsn string,
	slot string,
	opts ...Option,
) (*LiveDecoder, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, types.ErrReaderRequired
	}
	if strings.TrimSpace(slot) == "" {
		return nil, types.ErrSlotRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres live connect: %w", err)
	}
	closeConn := func() error {
		return conn.Close(context.Background())
	}
	return newLiveDecoder(ctx, source, pgxLiveQueryer{conn: conn}, closeConn, slot, opts...), nil
}

func newLiveDecoder(
	ctx context.Context,
	source dblog.Source,
	queryer liveQueryer,
	close func() error,
	slot string,
	opts ...Option,
) *LiveDecoder {
	if ctx == nil {
		ctx = context.Background()
	}
	if source.Driver == "" {
		source.Driver = types.Driver
	}
	if source.Name == "" {
		source.Name = slot
	}
	cfg := options{pollInterval: defaultLivePollInterval}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &LiveDecoder{
		ctx:           ctx,
		source:        source,
		queryer:       queryer,
		close:         close,
		slot:          slot,
		plugins:       cfg.eventPlugins,
		startPosition: cfg.startPosition,
		pollInterval:  cfg.pollInterval,
	}
}

func (d *LiveDecoder) Events() iter.Seq2[dblog.Event, error] {
	return func(yield func(dblog.Event, error) bool) {
		for d != nil && d.queryer != nil && d.ctx.Err() == nil {
			rows, err := d.queryer.Query(d.ctx, liveChangesQuery, d.slot)
			if err != nil {
				if d.ctx.Err() != nil || errors.Is(err, context.Canceled) {
					return
				}
				yield(nil, fmt.Errorf("postgres live query changes: %w", err))
				return
			}

			saw, ok := d.yieldRows(rows, yield)
			rows.Close()
			if !ok {
				return
			}
			if !saw && !d.waitForNextPoll() {
				return
			}
		}
	}
}

func (d *LiveDecoder) yieldRows(rows liveRows, yield func(dblog.Event, error) bool) (bool, bool) {
	var saw bool
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			yield(nil, fmt.Errorf("postgres live scan change: %w", err))
			return saw, false
		}
		d.position++
		if d.position <= d.startPosition || strings.TrimSpace(line) == "" {
			continue
		}
		event, err := parseLine(d.source, d.position, line, d.plugins)
		if err != nil {
			yield(nil, err)
			return saw, false
		}
		saw = true
		if !yield(event, nil) {
			return saw, false
		}
	}
	if err := rows.Err(); err != nil {
		if d.ctx.Err() != nil {
			return saw, false
		}
		yield(nil, fmt.Errorf("postgres live read changes: %w", err))
		return saw, false
	}
	return saw, true
}

func (d *LiveDecoder) waitForNextPoll() bool {
	if d.pollInterval <= 0 {
		return d.ctx.Err() == nil
	}
	timer := time.NewTimer(d.pollInterval)
	defer timer.Stop()
	select {
	case <-d.ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (d *LiveDecoder) Close() error {
	if d == nil || d.close == nil {
		return nil
	}
	return d.close()
}

var _ dblog.Decoder[dblog.Event] = (*LiveDecoder)(nil)
