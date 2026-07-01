package postgres

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
)

const postgresFixturePath = "testdata/test_decoding.log"

func TestFixtureBackedLogicalDecoding(t *testing.T) {
	path := requirePostgresFixture(t)

	decoder := openPostgresFixture(t, path)
	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if change, ok := event.Body().(Change); ok {
			if change.Schema != "public" || change.Table != "users" {
				t.Fatalf("change target = %#v", change)
			}
		}
		counts[event.Kind()]++
	}

	for _, kind := range []string{KindBegin, OperationInsert, OperationUpdate, OperationDelete, KindCommit} {
		if counts[kind] == 0 {
			t.Fatalf("fixture has no %s events: %v", kind, counts)
		}
	}

	decoder = openPostgresFixture(t, path)
	var deletes int
	for event, err := range dblog.Filter(decoder.Events(), dblog.ByKind(OperationDelete)) {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != OperationDelete {
			t.Fatalf("filtered event kind = %s", event.Kind())
		}
		deletes++
	}
	if deletes == 0 {
		t.Fatal("filtered delete events are empty")
	}

	decoder = openPostgresFixture(t, path)
	var flashbacks int
	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		sql := op.(string)
		if !strings.Contains(sql, "public.users") {
			t.Fatalf("flashback SQL = %s", sql)
		}
		flashbacks++
	}
	if flashbacks == 0 {
		t.Fatal("fixture produced no flashback SQL")
	}
}

func requirePostgresFixture(t *testing.T) string {
	t.Helper()
	if _, err := os.Stat(postgresFixturePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("test fixture %s not found; run testdata/generate_postgres_logical.sh", postgresFixturePath)
		}
		t.Fatal(err)
	}
	return postgresFixturePath
}

func openPostgresFixture(t *testing.T, path string) dblog.Decoder[dblog.Event] {
	t.Helper()
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithPath(path),
		dblog.WithSource(dblog.Source{Name: "test_decoding"}),
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
