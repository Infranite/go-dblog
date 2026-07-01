package mongo

import (
	"errors"
	"os"
	"testing"

	"github.com/Infranite/go-dblog"
)

const mongoFixturePath = "testdata/oplog.jsonl"

func TestFixtureBackedOplogDecoding(t *testing.T) {
	path := requireMongoFixture(t)

	decoder := openMongoFixture(t, path)
	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		change := event.Body().(Change)
		if change.Database != "dblog_ci" || change.Collection != "users" {
			t.Fatalf("change namespace = %#v", change)
		}
		counts[event.Kind()]++
	}

	for _, kind := range []string{OperationInsert, OperationUpdate, OperationDelete} {
		if counts[kind] == 0 {
			t.Fatalf("fixture has no %s events: %v", kind, counts)
		}
	}

	decoder = openMongoFixture(t, path)
	var inserts int
	for event, err := range dblog.Filter(decoder.Events(), dblog.ByKind(OperationInsert)) {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != OperationInsert {
			t.Fatalf("filtered event kind = %s", event.Kind())
		}
		inserts++
	}
	if inserts == 0 {
		t.Fatal("filtered insert events are empty")
	}

	decoder = openMongoFixture(t, path)
	var flashbacks int
	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		command := op.(Command)
		if command.Database != "dblog_ci" || command.Collection != "users" {
			t.Fatalf("flashback command = %#v", command)
		}
		flashbacks++
	}
	if flashbacks == 0 {
		t.Fatal("fixture produced no flashback commands")
	}
}

func requireMongoFixture(t *testing.T) string {
	t.Helper()
	if _, err := os.Stat(mongoFixturePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("test fixture %s not found; run testdata/generate_mongo_oplog.sh", mongoFixturePath)
		}
		t.Fatal(err)
	}
	return mongoFixturePath
}

func openMongoFixture(t *testing.T, path string) dblog.Decoder[dblog.Event] {
	t.Helper()
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithPath(path),
		dblog.WithSource(dblog.Source{Name: "oplog"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})
	return decoder
}
