package backend

import (
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
)

func TestRegisterOpensRedisDecoder(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}

	decoder, err := registry.Open(Driver, dblog.WithReader(strings.NewReader("*1\r\n$4\r\nINCR\r\n")))
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
		if event.SourceDriver() != Driver || event.Kind() != "incr" {
			t.Fatalf("event = %s/%s", event.SourceDriver(), event.Kind())
		}
		return
	}
	t.Fatal("no events")
}
