package backend

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Infranite/go-dblog"
)

func TestRegisterOpensMySQLDecoder(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join("..", "test", "testdata", "mysql-bin.000004")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("test binlog %s not found", path)
		}
		t.Fatal(err)
	}

	decoder, err := registry.Open(Driver, dblog.WithPath(path))
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
		if event.SourceDriver() != Driver {
			t.Fatalf("driver = %s", event.SourceDriver())
		}
		if event.Kind() != "FORMAT_DESCRIPTION_EVENT" {
			t.Fatalf("kind = %s", event.Kind())
		}
		return
	}
	t.Fatal("no events")
}

func TestIsMySQLDSN(t *testing.T) {
	if !isMySQLDSN("mysql://root@127.0.0.1:3306/") {
		t.Fatal("mysql URL was not detected")
	}
	if isMySQLDSN("testdata/mysql-bin.000004") {
		t.Fatal("file path was detected as mysql DSN")
	}
}

func TestDSNWithStartFile(t *testing.T) {
	got := dsnWithStartFile("mysql://root@127.0.0.1:3306/?server_id=7", "mysql-bin.000001")
	want := "mysql://root@127.0.0.1:3306/?file=mysql-bin.000001&server_id=7"
	if got != want {
		t.Fatalf("dsn = %q, want %q", got, want)
	}
	got = dsnWithStartFile("mysql://root@127.0.0.1:3306/?binlog=mysql-bin.000002", "mysql-bin.000001")
	if got != "mysql://root@127.0.0.1:3306/?binlog=mysql-bin.000002" {
		t.Fatalf("dsn with explicit file = %q", got)
	}
}

func TestRegisterResumesAfterCheckpoint(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join("..", "test", "testdata", "mysql-bin.000004")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("test binlog %s not found", path)
		}
		t.Fatal(err)
	}

	firstDecoder, err := registry.Open(Driver, dblog.WithPath(path))
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
		dblog.WithPath(path),
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
		if dblog.PositionOf(event) == checkpoint.Position {
			t.Fatalf("resumed at checkpoint again: %#v", checkpoint.Position)
		}
		if event.Kind() == "FORMAT_DESCRIPTION_EVENT" {
			t.Fatalf("resumed at first event again: %s", event.Kind())
		}
		return
	}
	t.Fatal("no resumed events")
}
