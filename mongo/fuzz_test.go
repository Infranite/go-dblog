package mongo

import (
	"testing"

	"github.com/Infranite/go-dblog"
)

func FuzzParseLine(f *testing.F) {
	f.Add(`{"op":"i","ns":"app.users","o":{"_id":1,"name":"Ada"}}`)
	f.Add(`{"operationType":"delete","ns":{"db":"app","coll":"users"},"documentKey":{"_id":1}}`)
	f.Add(`{"operationType":""}`)
	f.Add(`not json`)

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
