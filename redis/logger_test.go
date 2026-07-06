package redis

import (
	"testing"

	"github.com/Infranite/go-dblog"
)

func TestPackageLoggerIsIndependent(t *testing.T) {
	if Log == dblog.Log {
		t.Fatal("redis logger shares the root logger slot")
	}

	rootLevel := dblog.Log.Level()
	oldLevel := Log.Level()
	t.Cleanup(func() {
		Log.SetLevel(oldLevel)
	})

	Log.SetLevel(dblog.LevelError)
	if got := Log.Level(); got != dblog.LevelError {
		t.Fatalf("redis log level = %s, want %s", got, dblog.LevelError)
	}
	if got := dblog.Log.Level(); got != rootLevel {
		t.Fatalf("root log level changed to %s, want %s", got, rootLevel)
	}
}
