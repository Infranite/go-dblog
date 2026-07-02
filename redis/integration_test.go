package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Infranite/go-dblog"
)

const redisFixturePath = "testdata/appendonly.aof"
const redisLiveTimeout = 15 * time.Second

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
	var sawSafeReverse bool
	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		command := op.(Command)
		if len(command.Args) == 0 {
			t.Fatalf("flashback command = %#v", command)
		}
		switch command.Name {
		case CommandHDel, CommandSRem:
			t.Fatalf("state-dependent flashback command = %#v", command)
		case CommandLPop, CommandRPop, CommandIncr, CommandDecr, CommandIncrBy, CommandDecrBy:
			sawSafeReverse = true
		}
		flashbacks++
	}
	if flashbacks == 0 {
		t.Fatal("fixture produced no flashback commands")
	}
	if !sawSafeReverse {
		t.Fatal("fixture produced no safe Redis flashback commands")
	}
}

func TestLiveReplicationStream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live integration test in short mode")
	}
	dsn := os.Getenv("DBLOG_REDIS_LIVE_DSN")
	addr := os.Getenv("DBLOG_REDIS_LIVE_ADDR")
	if dsn == "" || addr == "" {
		t.Skip("set DBLOG_REDIS_LIVE_DSN and DBLOG_REDIS_LIVE_ADDR to run live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), redisLiveTimeout)
	defer cancel()

	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithContext(ctx),
		dblog.WithDSN(dsn),
		dblog.WithSource(dblog.Source{Name: "replication"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	writeErr := make(chan error, 1)
	go func() {
		time.Sleep(500 * time.Millisecond)
		for _, command := range [][]string{
			{"SET", "live:key", "value"},
			{"INCR", "live:counter"},
			{"LPUSH", "live:queue", "job-1"},
		} {
			if err := sendRedisCommand(addr, command...); err != nil {
				writeErr <- err
				return
			}
		}
		writeErr <- nil
	}()

	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		counts[event.Kind()]++
		if counts["set"] > 0 && counts[CommandIncr] > 0 && counts[CommandLPush] > 0 {
			cancel()
		}
	}
	if err := <-writeErr; err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"set", CommandIncr, CommandLPush} {
		if counts[kind] == 0 {
			t.Fatalf("live reader has no %s commands: %v", kind, counts)
		}
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

func sendRedisCommand(addr string, parts ...string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	var raw strings.Builder
	raw.WriteString(fmt.Sprintf("*%d\r\n", len(parts)))
	for _, part := range parts {
		raw.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(part), part))
	}
	if _, err := conn.Write([]byte(raw.String())); err != nil {
		return err
	}
	reply := make([]byte, 1)
	if _, err := conn.Read(reply); err != nil {
		return err
	}
	if reply[0] == '-' {
		return errors.New("redis command failed")
	}
	return nil
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
