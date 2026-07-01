package decoder

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/Infranite/go-dblog/mysql/common"
	"github.com/Infranite/go-dblog/mysql/decode/events"
	"github.com/Infranite/go-dblog/mysql/decode/events/types"
)

const testBinlogPath = "../../test/testdata/mysql-bin.000004"

func requireTestBinlog(t *testing.T) string {
	t.Helper()
	if _, err := os.Stat(testBinlogPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("test binlog %s not found; run mysql/test/testdata/generate_mysql_binlog.sh", testBinlogPath)
		}
		t.Fatal(err)
	}
	return testBinlogPath
}

func closeDecoder(t *testing.T, fileDecoder *BinFileDecoder) {
	t.Helper()
	if err := fileDecoder.Close(); err != nil {
		t.Error(err)
	}
}

func TestNewBinFileDecoderWithOptions(t *testing.T) {
	t.Parallel()

	fileDecoder, err := NewBinFileDecoder(requireTestBinlog(t), WithStartPos(0))
	if err != nil {
		t.Fatal(err)
	}
	defer closeDecoder(t, fileDecoder)
}

func TestWalkEventSkipsBodyBeforeStartPos(t *testing.T) {
	t.Parallel()

	startPos := firstRowsEventStartPos(t)
	fileDecoder, err := NewBinFileDecoder(requireTestBinlog(t), WithStartPos(startPos))
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

	if len(eventTypes) == 0 {
		t.Fatal("got no events")
	}
	if eventTypes[0] != common.FormatDescriptionEvent {
		t.Fatalf("first event = %s, want FORMAT_DESCRIPTION_EVENT", common.EventTypeName(eventTypes[0]))
	}
	if !containsRowsEvent(eventTypes[1:]) {
		t.Fatalf("got %v event types after start pos, want a rows event", eventTypes)
	}
}

func TestEventsIteratorAndGenericBodyFilter(t *testing.T) {
	t.Parallel()

	fileDecoder, err := NewBinFileDecoder(requireTestBinlog(t))
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

	fileDecoder, err := NewBinFileDecoder(requireTestBinlog(t))
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

	fileDecoder, err := NewBinFileDecoder(requireTestBinlog(t))
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

func firstRowsEventStartPos(t *testing.T) int64 {
	t.Helper()
	fileDecoder, err := NewBinFileDecoder(requireTestBinlog(t))
	if err != nil {
		t.Fatal(err)
	}
	defer closeDecoder(t, fileDecoder)

	for event, err := range fileDecoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if isRowsEvent(event.Header.EventType) {
			return event.Header.LogPos - event.Header.EventSize
		}
	}
	t.Fatal("rows event not found")
	return 0
}

func containsRowsEvent(eventTypes []uint8) bool {
	for _, eventType := range eventTypes {
		if isRowsEvent(eventType) {
			return true
		}
	}
	return false
}

func isRowsEvent(eventType uint8) bool {
	switch eventType {
	case common.WriteRowsEventV0, common.UpdateRowsEventV0, common.DeleteRowsEventV0,
		common.WriteRowsEventV1, common.UpdateRowsEventV1, common.DeleteRowsEventV1,
		common.WriteRowsEventV2, common.UpdateRowsEventV2, common.DeleteRowsEventV2,
		common.PartialUpdateRowsEvent:
		return true
	default:
		return false
	}
}
