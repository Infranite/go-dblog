package mongo

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
)

var benchmarkMongoEvent Event

func TestParseLineParsesOplogInsert(t *testing.T) {
	event, err := ParseLine(dblog.Source{Name: "oplog"}, 3, `{"op":"i","ns":"app.users","o":{"_id":1,"name":"Ada"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if event.SourceDriver() != Driver || event.PositionString() != "3" || event.Kind() != OperationInsert {
		t.Fatalf("event = %s/%s/%s", event.SourceDriver(), event.PositionString(), event.Kind())
	}
	change := event.Body().(Change)
	if change.Database != "app" || change.Collection != "users" {
		t.Fatalf("namespace = %#v", change)
	}
	if change.Document["name"] != "Ada" || change.DocumentKey["_id"] != json.Number("1") {
		t.Fatalf("change = %#v", change)
	}
}

func BenchmarkParseLine(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event, err := ParseLine(dblog.Source{Name: "oplog"}, i, `{"op":"i","ns":"app.users","o":{"_id":1,"name":"Ada"}}`)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkMongoEvent = event
	}
}

func TestBackendStreamsEventsAndFlashbacks(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithSource(dblog.Source{Name: "changes"}),
		dblog.WithReader(strings.NewReader(strings.Join([]string{
			`{"operationType":"insert","ns":{"db":"app","coll":"users"},"documentKey":{"_id":1},"fullDocument":{"_id":1,"name":"Ada"}}`,
			`{"operationType":"delete","ns":{"db":"app","coll":"users"},"documentKey":{"_id":2}}`,
		}, "\n"))),
	)
	if err != nil {
		t.Fatal(err)
	}
	streamDecoder := decoder
	t.Cleanup(func() {
		if err := streamDecoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	var kinds []string
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		kinds = append(kinds, event.Kind())
	}
	if strings.Join(kinds, ",") != OperationInsert+","+OperationDelete {
		t.Fatalf("kinds = %v", kinds)
	}

	decoder, err = registry.Open(Driver, dblog.WithReader(strings.NewReader(`{"operationType":"insert","ns":{"db":"app","coll":"users"},"documentKey":{"_id":1},"fullDocument":{"_id":1,"name":"Ada"}}`+"\n")))
	if err != nil {
		t.Fatal(err)
	}
	flashbackDecoder := decoder
	t.Cleanup(func() {
		if err := flashbackDecoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	var got []any
	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, op)
	}
	want := Command{
		Operation:  OperationDelete,
		Database:   "app",
		Collection: "users",
		Filter:     map[string]any{"_id": json.Number("1")},
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("flashbacks = %#v", got)
	}
}

func TestParseLineFlashbackRestoresUpdateBeforeImage(t *testing.T) {
	line := strings.Join([]string{
		`{"operationType":"update","ns":{"db":"app","coll":"users"},`,
		`"documentKey":{"_id":1},`,
		`"fullDocument":{"_id":1,"name":"Grace","role":"admin"},`,
		`"fullDocumentBeforeChange":{"_id":1,"name":"Ada","role":"reader"},`,
		`"updateDescription":{"updatedFields":{"name":"Grace","role":"admin"},"removedFields":[]}}`,
	}, "")
	event, err := ParseLine(dblog.Source{Name: "changes"}, 1, line)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := event.Reverse()
	if !ok {
		t.Fatal("expected flashback command")
	}
	want := Command{
		Operation:  CommandReplace,
		Database:   "app",
		Collection: "users",
		Filter:     map[string]any{"_id": json.Number("1")},
		Document: map[string]any{
			"_id":  json.Number("1"),
			"name": "Ada",
			"role": "reader",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flashback = %#v, want %#v", got, want)
	}
}

func TestParseLineFlashbackCopiesMutableMaps(t *testing.T) {
	line := strings.Join([]string{
		`{"operationType":"update","ns":{"db":"app","coll":"users"},`,
		`"documentKey":{"_id":1},`,
		`"fullDocumentBeforeChange":{"_id":1,"name":"Ada","profile":{"tier":"free"}}}`,
	}, "")
	event, err := ParseLine(dblog.Source{Name: "changes"}, 1, line)
	if err != nil {
		t.Fatal(err)
	}

	first, ok := event.Reverse()
	if !ok {
		t.Fatal("expected flashback command")
	}
	firstCommand := first.(Command)
	firstCommand.Filter["_id"] = json.Number("2")
	firstCommand.Document["name"] = "Grace"
	firstCommand.Document["profile"].(map[string]any)["tier"] = "paid"

	second, ok := event.Reverse()
	if !ok {
		t.Fatal("expected flashback command")
	}
	secondCommand := second.(Command)
	if secondCommand.Filter["_id"] != json.Number("1") ||
		secondCommand.Document["name"] != "Ada" ||
		secondCommand.Document["profile"].(map[string]any)["tier"] != "free" {
		t.Fatalf("flashback reused mutable maps: %#v", secondCommand)
	}
}

func TestParseLineRejectsUnsupportedOplogOperation(t *testing.T) {
	_, err := ParseLine(dblog.Source{}, 1, `{"op":"x","ns":"app.users"}`)
	if !errors.Is(err, ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want %v", err, ErrUnsupportedOperation)
	}
}

func TestParseLineRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{name: "invalid json", line: `{"op":"i","ns":"app.users","o":`},
		{name: "update description string", line: `{"operationType":"update","ns":{"db":"app","coll":"users"},"updateDescription":"bad"}`},
		{name: "update description array", line: `{"operationType":"update","ns":{"db":"app","coll":"users"},"updateDescription":[]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseLine(dblog.Source{}, 1, tt.line)
			if !errors.Is(err, ErrInvalidJSON) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidJSON)
			}
		})
	}
}

func TestBackendRequiresInput(t *testing.T) {
	_, err := Backend{}.Open(nilOptions{})
	if !errors.Is(err, ErrReaderRequired) {
		t.Fatalf("err = %v, want %v", err, ErrReaderRequired)
	}
}
