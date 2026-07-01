package decoder

import (
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
)

type replacePlugin struct{}

func (replacePlugin) Name() string { return "replace" }

func (replacePlugin) Match(raw map[string]any) bool {
	return raw["operationType"] == "replace"
}

func (replacePlugin) Apply(change *types.Change) error {
	change.Operation = types.OperationUpdate
	return nil
}

func TestDecoderAppliesEventPlugins(t *testing.T) {
	decoder := NewDecoder(
		dblog.Source{Name: "changes"},
		strings.NewReader(`{"operationType":"replace","ns":{"db":"app","coll":"users"},"documentKey":{"_id":1},"fullDocument":{"_id":1}}`+"\n"),
		nil,
		WithEventPlugins(replacePlugin{}),
	)

	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != types.OperationUpdate {
			t.Fatalf("kind = %s", event.Kind())
		}
		return
	}
	t.Fatal("no events")
}
