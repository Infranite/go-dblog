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
