package decoder

import (
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

type messagePlugin struct{}

func (messagePlugin) Name() string { return "message" }

func (messagePlugin) Match(line string) bool {
	return strings.HasPrefix(line, "message ")
}

func (messagePlugin) Decode(source dblog.Source, position int, line string) (types.Event, error) {
	return types.NewEvent(source, position, []byte(line), "message", line), nil
}

func TestDecoderAppliesEventPlugins(t *testing.T) {
	decoder := NewDecoder(
		dblog.Source{Name: "slot"},
		strings.NewReader("message hello\n"),
		nil,
		WithEventPlugins(messagePlugin{}),
	)

	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != "message" {
			t.Fatalf("kind = %s", event.Kind())
		}
		return
	}
	t.Fatal("no events")
}
