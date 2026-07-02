package decoder

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

type fakeLiveQueryer struct {
	query string
	args  []any
	rows  liveRows
}

func (q *fakeLiveQueryer) Query(_ context.Context, query string, args ...any) (liveRows, error) {
	q.query = query
	q.args = append([]any(nil), args...)
	return q.rows, nil
}

type fakeLiveRows struct {
	lines []string
	err   error
	pos   int
}

func (r *fakeLiveRows) Next() bool {
	return r.pos < len(r.lines)
}

func (r *fakeLiveRows) Scan(dest ...any) error {
	*(dest[0].(*string)) = r.lines[r.pos]
	r.pos++
	return nil
}

func (r *fakeLiveRows) Err() error { return r.err }

func (r *fakeLiveRows) Close() {}

func TestLiveDecoderPollsLogicalSlot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	queryer := &fakeLiveQueryer{rows: &fakeLiveRows{lines: []string{
		"BEGIN 42",
		"table public.users: INSERT: id[integer]:1 name[text]:'Ada'",
	}}}
	decoder := newLiveDecoder(
		ctx,
		dblog.Source{Name: "dblog_ci_slot"},
		queryer,
		nil,
		"dblog_ci_slot",
		WithPollInterval(time.Nanosecond),
	)

	var kinds []string
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		kinds = append(kinds, event.Kind())
		if len(kinds) == 2 {
			cancel()
		}
	}

	if strings.Join(kinds, ",") != strings.Join([]string{types.KindBegin, types.OperationInsert}, ",") {
		t.Fatalf("kinds = %v", kinds)
	}
	if !strings.Contains(queryer.query, "pg_logical_slot_get_changes") {
		t.Fatalf("query = %q", queryer.query)
	}
	if len(queryer.args) != 1 || queryer.args[0] != "dblog_ci_slot" {
		t.Fatalf("args = %#v", queryer.args)
	}
}

func TestLiveDecoderStopsAfterRowError(t *testing.T) {
	wantErr := errors.New("row read")
	decoder := newLiveDecoder(
		context.Background(),
		dblog.Source{Name: "dblog_ci_slot"},
		nil,
		nil,
		"dblog_ci_slot",
	)

	var gotErr error
	_, ok := decoder.yieldRows(&fakeLiveRows{err: wantErr}, func(_ dblog.Event, err error) bool {
		gotErr = err
		return true
	})
	if ok {
		t.Fatal("yieldRows continued after row error")
	}
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("err = %v, want %v", gotErr, wantErr)
	}
}
