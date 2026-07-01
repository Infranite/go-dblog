package decoder

import (
	"testing"

	"github.com/Infranite/go-dblog/mysql/decode/events/types"
)

func TestDblogDecoderEvents(t *testing.T) {
	fileDecoder, err := NewBinFileDecoder(requireTestBinlog(t))
	if err != nil {
		t.Fatal(err)
	}
	defer fileDecoder.Close()

	decoder := WrapDblogDecoder(Source{Name: "mysql-bin.000004"}, fileDecoder)
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if event.SourceDriver() != "mysql" || event.SourceName() != "mysql-bin.000004" {
			t.Fatalf("source = %s/%s", event.SourceDriver(), event.SourceName())
		}
		if event.Kind() != "FORMAT_DESCRIPTION_EVENT" {
			t.Fatalf("kind = %s", event.Kind())
		}
		if event.PositionDriver() != "mysql" || event.PositionString() == "" {
			t.Fatalf("position = %s/%s", event.PositionDriver(), event.PositionString())
		}
		if _, ok := event.Body().(*types.FmtDescEvent); !ok {
			t.Fatalf("body = %T", event.Body())
		}
		if len(event.Raw()) == 0 {
			t.Fatal("raw event is empty")
		}
		return
	}
	t.Fatal("no events")
}

func TestDblogBodiesFiltersTypedBody(t *testing.T) {
	decoder, err := NewDblogDecoder(requireTestBinlog(t))
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		body, ok := event.Body().(*types.QueryEvent)
		if !ok {
			continue
		}
		if body.Query == "" {
			t.Fatal("query event has empty query")
		}
		return
	}
	t.Fatal("query event not found")
}
