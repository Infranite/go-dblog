package backend

import (
	"errors"
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
)

func TestRegisterOpensMongoDecoder(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}

	decoder, err := registry.Open(Driver, dblog.WithReader(strings.NewReader(`{"op":"i","ns":"app.users","o":{"_id":1}}`+"\n")))
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
		if event.SourceDriver() != Driver || event.Kind() != "insert" {
			t.Fatalf("event = %s/%s", event.SourceDriver(), event.Kind())
		}
		return
	}
	t.Fatal("no events")
}

func TestOpenLiveDSNRequiresCollection(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}

	_, err := registry.Open(Driver, dblog.WithDSN("mongodb://127.0.0.1:27017"))
	if !errors.Is(err, types.ErrCollectionRequired) {
		t.Fatalf("err = %v, want %v", err, types.ErrCollectionRequired)
	}
}

func TestIsMongoDSN(t *testing.T) {
	if !isMongoDSN("mongodb://127.0.0.1:27017") {
		t.Fatal("mongo URL was not detected")
	}
	if !isMongoDSN("mongodb+srv://cluster.example.net") {
		t.Fatal("mongo SRV URL was not detected")
	}
	if isMongoDSN("testdata/oplog.jsonl") {
		t.Fatal("file path was detected as mongo DSN")
	}
}

func TestRegisterResumesAfterCheckpoint(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}

	input := strings.Join([]string{
		`{"op":"i","ns":"app.users","o":{"_id":1}}`,
		`{"op":"d","ns":"app.users","o":{"_id":1}}`,
	}, "\n") + "\n"

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
		if event.Kind() != "delete" || event.PositionString() != "2" {
			t.Fatalf("event = %s at %s", event.Kind(), event.PositionString())
		}
		return
	}
	t.Fatal("no resumed events")
}
