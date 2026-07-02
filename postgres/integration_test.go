package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Infranite/go-dblog"
)

const postgresFixturePath = "testdata/test_decoding.log"
const postgresLiveTimeout = 15 * time.Second

func TestFixtureBackedLogicalDecoding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fixture-backed integration test in short mode")
	}
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
	var updateFlashbacks int
	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		sql := op.(string)
		if !strings.Contains(sql, "public.users") {
			t.Fatalf("flashback SQL = %s", sql)
		}
		if strings.HasPrefix(sql, "UPDATE public.users SET ") {
			updateFlashbacks++
		}
		flashbacks++
	}
	if flashbacks == 0 {
		t.Fatal("fixture produced no flashback SQL")
	}
	if updateFlashbacks == 0 {
		t.Fatal("fixture produced no update flashback SQL")
	}
}

func TestLiveLogicalDecoding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live integration test in short mode")
	}
	dsn := os.Getenv("DBLOG_POSTGRES_LIVE_DSN")
	slot := os.Getenv("DBLOG_POSTGRES_LIVE_SLOT")
	if dsn == "" || slot == "" {
		t.Skip("set DBLOG_POSTGRES_LIVE_DSN and DBLOG_POSTGRES_LIVE_SLOT to run live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), postgresLiveTimeout)
	defer cancel()

	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithContext(ctx),
		dblog.WithDSN(dsn),
		dblog.WithSource(dblog.Source{Name: slot}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		counts[event.Kind()]++
		if counts[OperationInsert] > 0 && counts[OperationUpdate] > 0 && counts[OperationDelete] > 0 {
			cancel()
		}
	}

	for _, kind := range []string{KindBegin, OperationInsert, OperationUpdate, OperationDelete, KindCommit} {
		if counts[kind] == 0 {
			t.Fatalf("live reader has no %s events: %v", kind, counts)
		}
	}
}

func TestWireLogicalReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wire integration test in short mode")
	}
	dsn := os.Getenv("DBLOG_POSTGRES_WIRE_DSN")
	slot := os.Getenv("DBLOG_POSTGRES_WIRE_SLOT")
	if dsn == "" || slot == "" {
		t.Skip("set DBLOG_POSTGRES_WIRE_DSN and DBLOG_POSTGRES_WIRE_SLOT to run wire test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), postgresLiveTimeout)
	defer cancel()

	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithContext(ctx),
		dblog.WithDSN(dsn),
		dblog.WithSource(dblog.Source{Name: slot}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		counts[event.Kind()]++
		if counts[KindBegin] > 0 && counts[OperationInsert] > 0 &&
			counts[OperationUpdate] > 0 && counts[OperationDelete] > 0 &&
			counts[KindCommit] > 0 {
			cancel()
		}
	}

	for _, kind := range []string{KindBegin, OperationInsert, OperationUpdate, OperationDelete, KindCommit} {
		if counts[kind] == 0 {
			t.Fatalf("wire reader has no %s events: %v", kind, counts)
		}
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
