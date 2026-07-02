package backend

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

type testOptions struct {
	source dblog.Source
	dsn    string
}

func (o testOptions) Source() dblog.Source { return o.source }
func (o testOptions) Path() string         { return "" }
func (o testOptions) DSN() string          { return o.dsn }
func (o testOptions) Reader() io.Reader    { return nil }

func TestRegisterOpensPostgresDecoder(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}

	decoder, err := registry.Open(Driver, dblog.WithReader(strings.NewReader("BEGIN 42\n")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if event.SourceDriver() != Driver || event.Kind() != "begin" {
			t.Fatalf("event = %s/%s", event.SourceDriver(), event.Kind())
		}
		return
	}
	t.Fatal("no events")
}

func TestOpenLiveDSNRequiresSlot(t *testing.T) {
	_, err := Backend{}.Open(testOptions{dsn: "postgres://postgres:postgres@127.0.0.1/postgres?sslmode=disable"})
	if !errors.Is(err, types.ErrSlotRequired) {
		t.Fatalf("err = %v, want %v", err, types.ErrSlotRequired)
	}
}

func TestIsPostgresDSN(t *testing.T) {
	if !isPostgresDSN("postgres://postgres@127.0.0.1/postgres") {
		t.Fatal("postgres URL was not detected")
	}
	if isPostgresDSN("testdata/test_decoding.log") {
		t.Fatal("file path was detected as postgres DSN")
	}
}

func TestRegisterResumesAfterCheckpoint(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}

	input := "BEGIN 42\nCOMMIT 42\n"

	firstDecoder, err := registry.Open(Driver, dblog.WithReader(strings.NewReader(input)))
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint dblog.Checkpoint
	for event, err := range firstDecoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		checkpoint = dblog.CheckpointOf(event)
		break
	}
	if err := firstDecoder.Close(); err != nil {
		t.Fatal(err)
	}

	decoder, err := registry.Open(Driver,
		dblog.WithReader(strings.NewReader(input)),
		dblog.WithCheckpoint(checkpoint),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != "commit" || event.PositionString() != "2" {
			t.Fatalf("event = %s at %s", event.Kind(), event.PositionString())
		}
		return
	}
	t.Fatal("no resumed events")
}
