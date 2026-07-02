package decoder

import (
	"context"
	"errors"
	"testing"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeLiveWatcher struct {
	cursor liveCursor
	err    error
}

func (w fakeLiveWatcher) Watch(context.Context) (liveCursor, error) {
	return w.cursor, w.err
}

type fakeLiveCursor struct {
	changes []map[string]any
	err     error
	closed  bool
	index   int
}

func (c *fakeLiveCursor) Next(context.Context) bool {
	if c.index >= len(c.changes) {
		return false
	}
	c.index++
	return true
}

func (c *fakeLiveCursor) Decode(v any) error {
	change := v.(*map[string]any)
	*change = c.changes[c.index-1]
	return nil
}

func (c *fakeLiveCursor) Err() error { return c.err }

func (c *fakeLiveCursor) Close(context.Context) error {
	c.closed = true
	return nil
}

func TestLiveDecoderStreamsChangeStreamEvents(t *testing.T) {
	cursor := &fakeLiveCursor{changes: []map[string]any{{
		"operationType": "insert",
		"ns": map[string]any{
			"db":   "app",
			"coll": "users",
		},
		"documentKey":  map[string]any{"_id": int32(1)},
		"fullDocument": map[string]any{"_id": int32(1), "name": "Ada"},
	}}}
	decoder := newLiveDecoder(
		context.Background(),
		dblog.Source{Name: "app.users"},
		fakeLiveWatcher{cursor: cursor},
		nil,
		"app.users",
	)

	var got []dblog.Event
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, event)
	}

	if len(got) != 1 {
		t.Fatalf("events = %d, want 1", len(got))
	}
	if got[0].SourceDriver() != types.Driver || got[0].Kind() != types.OperationInsert {
		t.Fatalf("event = %s/%s", got[0].SourceDriver(), got[0].Kind())
	}
	if !cursor.closed {
		t.Fatal("cursor was not closed")
	}
}

func TestLiveDecoderStopsAfterCursorError(t *testing.T) {
	wantErr := errors.New("cursor failed")
	decoder := newLiveDecoder(
		context.Background(),
		dblog.Source{Name: "app.users"},
		fakeLiveWatcher{cursor: &fakeLiveCursor{err: wantErr}},
		nil,
		"app.users",
	)

	for _, err := range decoder.Events() {
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
		return
	}
	t.Fatal("no error")
}

func TestParseRawAcceptsBSONDocuments(t *testing.T) {
	event, err := parseRaw(dblog.Source{Name: "app.users"}, 1, nil, map[string]any{
		"operationType": "insert",
		"ns":            bson.D{{Key: "db", Value: "app"}, {Key: "coll", Value: "users"}},
		"documentKey":   bson.D{{Key: "_id", Value: int32(1)}},
		"fullDocument":  bson.D{{Key: "_id", Value: int32(1)}, {Key: "name", Value: "Ada"}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	change := event.Body().(types.Change)
	if change.Database != "app" || change.Collection != "users" || change.DocumentKey["_id"] != int32(1) {
		t.Fatalf("change = %#v", change)
	}
}
