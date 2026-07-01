package decoder

import (
	"errors"
	"io"
	"testing"

	"github.com/Infranite/go-mysql-binlog/binlog/common"
	"github.com/Infranite/go-mysql-binlog/binlog/decode/events"
	"github.com/Infranite/go-mysql-binlog/binlog/decode/events/types"
)

const testBinlogPath = "../../../test/testdata/mysql-bin.000004"

func closeDecoder(t *testing.T, fileDecoder *BinFileDecoder) {
	t.Helper()
	if err := fileDecoder.Close(); err != nil {
		t.Error(err)
	}
}

func TestNewBinFileDecoderWithOptions(t *testing.T) {
	t.Parallel()

	fileDecoder, err := NewBinFileDecoder(testBinlogPath, WithStartPos(362))
	if err != nil {
		t.Fatal(err)
	}
	defer closeDecoder(t, fileDecoder)
}

func TestWalkEventSkipsBodyBeforeStartPos(t *testing.T) {
	t.Parallel()

	fileDecoder, err := NewBinFileDecoder(testBinlogPath, WithStartPos(362))
	if err != nil {
		t.Fatal(err)
	}
	defer closeDecoder(t, fileDecoder)

	var eventTypes []uint8
	err = fileDecoder.WalkEvent(func(event *events.Event) (bool, error) {
		eventTypes = append(eventTypes, event.Header.EventType)
		return len(eventTypes) < 3, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []uint8{
		common.FormatDescriptionEvent,
		common.WriteRowsEventV2,
		common.XIDEvent,
	}
	if len(eventTypes) != len(want) {
		t.Fatalf("got %v event types, want %v", eventTypes, want)
	}
	for i := range want {
		if eventTypes[i] != want[i] {
			t.Fatalf("eventTypes[%d] = %s, want %s", i, common.EventType2Str[eventTypes[i]], common.EventType2Str[want[i]])
		}
	}
}

func TestEventsIteratorAndGenericBodyFilter(t *testing.T) {
	t.Parallel()

	fileDecoder, err := NewBinFileDecoder(testBinlogPath)
	if err != nil {
		t.Fatal(err)
	}
	defer closeDecoder(t, fileDecoder)

	var queries []*types.QueryEvent
	for queryEvent, err := range EventBodies[*types.QueryEvent](fileDecoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		queries = append(queries, queryEvent)
	}

	if len(queries) == 0 {
		t.Fatal("got 0 query events, want at least 1")
	}
	for i, query := range queries {
		if query.Query == "" {
			t.Fatalf("queries[%d] is empty", i)
		}
	}
}

func TestRowsEventDecodesHeaderAndBitmaps(t *testing.T) {
	t.Parallel()

	fileDecoder, err := NewBinFileDecoder(testBinlogPath)
	if err != nil {
		t.Fatal(err)
	}
	defer closeDecoder(t, fileDecoder)

	var rows []*types.BinRowsEvent
	for rowsEvent, err := range EventBodies[*types.BinRowsEvent](fileDecoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, rowsEvent)
	}

	if len(rows) == 0 {
		t.Fatal("got 0 rows events, want at least 1")
	}
	if rows[0].TableID == 0 {
		t.Fatal("rows event table id is empty")
	}
	if rows[0].ColumnCount == 0 {
		t.Fatal("rows event column count is empty")
	}
	if len(rows[0].ColumnsBitmap1) == 0 {
		t.Fatal("rows event columns bitmap is empty")
	}
}

func TestEventsIteratorStopsAfterCallbackFalse(t *testing.T) {
	t.Parallel()

	fileDecoder, err := NewBinFileDecoder(testBinlogPath)
	if err != nil {
		t.Fatal(err)
	}
	defer closeDecoder(t, fileDecoder)

	var count int
	for _, err := range fileDecoder.Events() {
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
		count++
		break
	}

	if count != 1 {
		t.Fatalf("iterator yielded %d events, want 1", count)
	}
}
