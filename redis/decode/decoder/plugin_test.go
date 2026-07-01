package decoder

import (
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

type renamePlugin struct{}

func (renamePlugin) Name() string { return "rename" }

func (renamePlugin) Match(command types.Command) bool {
	return command.Name == "json.set"
}

func (renamePlugin) Apply(command *types.Command) error {
	command.Name = "jsonset"
	return nil
}

func TestDecoderAppliesCommandPlugins(t *testing.T) {
	decoder := NewDecoder(
		dblog.Source{Name: "appendonly.aof"},
		strings.NewReader("*2\r\n$8\r\nJSON.SET\r\n$5\r\nkey:1\r\n"),
		nil,
		WithCommandPlugins(renamePlugin{}),
	)

	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != "jsonset" {
			t.Fatalf("kind = %s", event.Kind())
		}
		return
	}
	t.Fatal("no events")
}
