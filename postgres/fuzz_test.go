package postgres

import (
	"testing"

	"github.com/Infranite/go-dblog"
)

func FuzzParseLine(f *testing.F) {
	f.Add("BEGIN 42")
	f.Add("COMMIT 42")
	f.Add("table public.users: INSERT: id[integer]:1 name[text]:'Ada'")
	f.Add("table public.users: DELETE: id[integer]:2 name[text]:'Grace'")
	f.Add("table : : ")
	f.Add("message unsupported")

	f.Fuzz(func(t *testing.T, line string) {
		event, err := ParseLine(dblog.Source{Name: "fuzz"}, 1, line)
		if err != nil {
			return
		}
		if event.SourceDriver() != Driver || event.PositionString() != "1" {
			t.Fatalf("event source/position = %s/%s", event.SourceDriver(), event.PositionString())
		}
		if event.Kind() == "" {
			t.Fatal("event kind is empty")
		}
		_, _ = event.Reverse()
	})
}
