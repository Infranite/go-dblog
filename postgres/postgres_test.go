package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
)

var benchmarkPostgresEvent Event

func TestParseLineParsesRecordChange(t *testing.T) {
	event, err := ParseLine(dblog.Source{Name: "slot"}, 7, "table public.users: INSERT: id[integer]:1 name[text]:'Ada Lovelace' active[boolean]:true")
	if err != nil {
		t.Fatal(err)
	}
	if event.SourceDriver() != Driver || event.SourceName() != "slot" {
		t.Fatalf("source = %s/%s", event.SourceDriver(), event.SourceName())
	}
	if event.PositionString() != "7" || event.Kind() != OperationInsert {
		t.Fatalf("position/kind = %s/%s", event.PositionString(), event.Kind())
	}
	change := event.Body().(Change)
	if change.Schema != "public" || change.Table != "users" || change.Operation != OperationInsert {
		t.Fatalf("change = %#v", change)
	}
	if len(change.Columns) != 3 || change.Columns[1].Value != "Ada Lovelace" || change.Columns[2].Value != true {
		t.Fatalf("columns = %#v", change.Columns)
	}
}

func BenchmarkParseLine(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event, err := ParseLine(dblog.Source{Name: "slot"}, i, "table public.users: INSERT: id[integer]:1 name[text]:'Ada Lovelace' active[boolean]:true")
		if err != nil {
			b.Fatal(err)
		}
		benchmarkPostgresEvent = event
	}
}

func TestBackendStreamsEventsAndFlashbacks(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithSource(dblog.Source{Name: "slot"}),
		dblog.WithReader(strings.NewReader(strings.Join([]string{
			"BEGIN 42",
			"table public.users: INSERT: id[integer]:1 name[text]:'Ada'",
			"table public.users: DELETE: id[integer]:2 name[text]:'Grace'",
			"COMMIT 42",
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
	if strings.Join(kinds, ",") != strings.Join([]string{KindBegin, OperationInsert, OperationDelete, KindCommit}, ",") {
		t.Fatalf("kinds = %v", kinds)
	}

	decoder, err = registry.Open(Driver, dblog.WithReader(strings.NewReader("table public.users: DELETE: id[integer]:2 name[text]:'Grace'\n")))
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
	if len(got) != 1 || got[0] != `INSERT INTO public.users (id, name) VALUES (2, 'Grace');` {
		t.Fatalf("flashbacks = %#v", got)
	}
}

func TestParseLineFlashbackRestoresUpdateOldKey(t *testing.T) {
	event, err := ParseLine(
		dblog.Source{Name: "slot"},
		1,
		"table public.users: UPDATE: old-key: id[integer]:1 name[text]:'Ada' new-tuple: id[integer]:2 name[text]:'Grace'",
	)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := event.Reverse()
	if !ok {
		t.Fatal("expected flashback SQL")
	}
	want := `UPDATE public.users SET id = 1, name = 'Ada' WHERE id = 2 AND name = 'Grace';`
	if got != want {
		t.Fatalf("flashback = %q, want %q", got, want)
	}
}

func TestParseLineSkipsUpdateFlashbackWithPartialOldKey(t *testing.T) {
	event, err := ParseLine(
		dblog.Source{Name: "slot"},
		1,
		"table public.users: UPDATE: old-key: id[integer]:1 new-tuple: id[integer]:2 name[text]:'Grace'",
	)
	if err != nil {
		t.Fatal(err)
	}

	if got, ok := event.Reverse(); ok {
		t.Fatalf("flashback = %#v, want none", got)
	}
}

func TestParseLineRejectsInvalidLogicalLine(t *testing.T) {
	_, err := ParseLine(dblog.Source{}, 1, "message prefix payload")
	if !errors.Is(err, ErrInvalidLine) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidLine)
	}
}

func TestBackendRequiresInput(t *testing.T) {
	_, err := Backend{}.Open(nilOptions{})
	if !errors.Is(err, ErrReaderRequired) {
		t.Fatalf("err = %v, want %v", err, ErrReaderRequired)
	}
}
