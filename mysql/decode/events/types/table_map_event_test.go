package types

import (
	"testing"

	"github.com/Infranite/go-dblog/mysql/common"
)

func TestTableMapEventKeepsOptionalMetadata(t *testing.T) {
	t.Parallel()

	context := NewEventContext()
	context.Description = testFmtDescEvent()

	data := []byte{
		1, 0, 0, 0, 0, 0, // table id
		0, 0, // flags
		1, 'd', 0, // schema
		1, 't', 0, // table
		1,                    // column count
		common.MySQLTypeLong, // column type
		0,                    // metadata length
		0,                    // null bitmap
		1, 1, 0xff,           // optional metadata TLV
	}

	body, err := new(TableMapEvent).Decode(WithData(data), WithContext(context))
	if err != nil {
		t.Fatal(err)
	}
	event := body.(*TableMapEvent)
	if got := event.OptionalMeta; len(got) != 3 || got[0] != 1 || got[1] != 1 || got[2] != 0xff {
		t.Fatalf("optional metadata = %v, want [1 1 255]", got)
	}
	if context.TableInfo[1] != event {
		t.Fatal("table map event was not stored in context")
	}
}
