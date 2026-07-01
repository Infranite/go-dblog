package redis

import (
	"errors"
	"os"
	"testing"

	"github.com/Infranite/go-dblog"
)

const redisFixturePath = "testdata/appendonly.aof"

func TestFixtureBackedAOFDecoding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fixture-backed integration test in short mode")
	}
	path := requireRedisFixture(t)

	decoder := openRedisFixture(t, path)
	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		counts[event.Kind()]++
	}

	for _, kind := range []string{CommandHSet, CommandSAdd, CommandLPush, CommandIncr} {
		if counts[kind] == 0 {
			t.Fatalf("fixture has no %s commands: %v", kind, counts)
		}
	}

	decoder = openRedisFixture(t, path)
	var hsets int
	for event, err := range dblog.Filter(decoder.Events(), dblog.ByKind(CommandHSet)) {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != CommandHSet {
			t.Fatalf("filtered event kind = %s", event.Kind())
		}
		hsets++
	}
	if hsets == 0 {
		t.Fatal("filtered hset commands are empty")
	}

	decoder = openRedisFixture(t, path)
	var flashbacks int
	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		command := op.(Command)
		if len(command.Args) == 0 {
			t.Fatalf("flashback command = %#v", command)
		}
		flashbacks++
	}
	if flashbacks == 0 {
		t.Fatal("fixture produced no flashback commands")
	}
}

func requireRedisFixture(t *testing.T) string {
	t.Helper()
	if _, err := os.Stat(redisFixturePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("test fixture %s not found; run testdata/generate_redis_aof.sh", redisFixturePath)
		}
		t.Fatal(err)
	}
	return redisFixturePath
}

func openRedisFixture(t *testing.T, path string) dblog.Decoder[dblog.Event] {
	t.Helper()
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithPath(path),
		dblog.WithSource(dblog.Source{Name: "appendonly.aof"}),
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
