package mysql

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Infranite/go-dblog"
	mysqlclient "github.com/go-mysql-org/go-mysql/client"
)

const mysqlLiveTimeout = 20 * time.Second

func TestLiveReplicationStream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live integration test in short mode")
	}
	dsn := os.Getenv("DBLOG_MYSQL_LIVE_DSN")
	addr := os.Getenv("DBLOG_MYSQL_LIVE_ADDR")
	user := os.Getenv("DBLOG_MYSQL_LIVE_USER")
	password := os.Getenv("DBLOG_MYSQL_LIVE_PASSWORD")
	if dsn == "" || addr == "" || user == "" {
		t.Skip("set DBLOG_MYSQL_LIVE_DSN, DBLOG_MYSQL_LIVE_ADDR, and DBLOG_MYSQL_LIVE_USER to run live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), mysqlLiveTimeout)
	defer cancel()

	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithContext(ctx),
		dblog.WithDSN(dsn),
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
		writeErr <- writeMySQLRows(addr, user, password)
	}()

	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		counts[event.Kind()]++
		if hasAnyKind(counts, "WRITE_ROWS_EVENT") &&
			hasAnyKind(counts, "UPDATE_ROWS_EVENT") &&
			hasAnyKind(counts, "DELETE_ROWS_EVENT") {
			cancel()
		}
	}
	if err := <-writeErr; err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []string{"WRITE_ROWS_EVENT", "UPDATE_ROWS_EVENT", "DELETE_ROWS_EVENT"} {
		if !hasAnyKind(counts, prefix) {
			t.Fatalf("live reader has no %s events: %v", prefix, counts)
		}
	}
}

func writeMySQLRows(addr, user, password string) error {
	conn, err := mysqlclient.Connect(addr, user, password, "")
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	for _, statement := range []string{
		"CREATE DATABASE IF NOT EXISTS dblog_live",
		"DROP TABLE IF EXISTS dblog_live.events",
		"CREATE TABLE dblog_live.events (id BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT, name VARCHAR(64) NOT NULL, amount INT NOT NULL) ENGINE=InnoDB",
		"INSERT INTO dblog_live.events(name, amount) VALUES ('alpha', 1)",
		"UPDATE dblog_live.events SET amount = 2 WHERE name = 'alpha'",
		"DELETE FROM dblog_live.events WHERE name = 'alpha'",
	} {
		if _, err := conn.Execute(statement); err != nil {
			return err
		}
	}
	return nil
}

func hasAnyKind(counts map[string]int, prefix string) bool {
	for kind, count := range counts {
		if count > 0 && strings.HasPrefix(kind, prefix) {
			return true
		}
	}
	return false
}
