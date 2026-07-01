package types

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/liipx/go-mysql-binlog/binlog/common"
)

func testFmtDescEvent() *FmtDescEvent {
	header := make([]byte, common.GTIDTaggedLogEvent)
	header[common.TableMapEvent-1] = 10
	header[common.WriteRowsEventV2-1] = 10
	header[common.UpdateRowsEventV2-1] = 10
	header[common.DeleteRowsEventV2-1] = 10
	return &FmtDescEvent{EventTypeHeader: header}
}

func TestRowsEventDecodesWriteRowsValues(t *testing.T) {
	t.Parallel()

	table := &TableMapEvent{
		TableID:       1,
		ColumnCount:   2,
		ColumnTypeDef: []byte{common.MySQLTypeLong, common.MySQLTypeVarchar},
		ColumnMetaDef: []uint16{0, 255},
	}
	tables := map[uint64]*TableMapEvent{1: table}

	data := []byte{
		1, 0, 0, 0, 0, 0, // table id
		0, 0, // flags
		2, 0, // v2 extra data length, no extra data
		2,           // column count
		0x03,        // columns-present-bitmap
		0x00,        // row null bitmap
		42, 0, 0, 0, // LONG
		2, 'h', 'i', // VARCHAR
	}

	event, err := decodeRowsEvent(data, testFmtDescEvent(), tables, common.WriteRowsEventV2)
	if err != nil {
		t.Fatal(err)
	}
	if len(event.Rows) != 1 {
		t.Fatalf("decoded %d rows, want 1", len(event.Rows))
	}
	row := event.Rows[0]
	if len(row) != 2 {
		t.Fatalf("decoded %d columns, want 2", len(row))
	}
	if got, ok := row[0].Value.(int64); !ok || got != 42 {
		t.Fatalf("row[0] = %#v, want int64(42)", row[0].Value)
	}
	if string(row[1].Raw) != "hi" {
		t.Fatalf("row[1] raw = %q, want hi", row[1].Raw)
	}
}

func TestRowsEventDecodesUpdateRowsBeforeAfterValues(t *testing.T) {
	t.Parallel()

	table := &TableMapEvent{
		TableID:       1,
		ColumnCount:   2,
		ColumnTypeDef: []byte{common.MySQLTypeLong, common.MySQLTypeVarchar},
		ColumnMetaDef: []uint16{0, 255},
	}
	tables := map[uint64]*TableMapEvent{1: table}

	data := []byte{
		1, 0, 0, 0, 0, 0, // table id
		0, 0, // flags
		2, 0, // v2 extra data length, no extra data
		2,          // column count
		0x01,       // before image only has column 0
		0x03,       // after image has column 0 and column 1
		0x00,       // before null bitmap
		1, 0, 0, 0, // before LONG
		0x02,       // after null bitmap: column 1 is NULL
		2, 0, 0, 0, // after LONG
	}

	event, err := decodeRowsEvent(data, testFmtDescEvent(), tables, common.UpdateRowsEventV2)
	if err != nil {
		t.Fatal(err)
	}
	if len(event.BeforeRows) != 1 || len(event.AfterRows) != 1 {
		t.Fatalf("decoded before=%d after=%d rows, want 1/1", len(event.BeforeRows), len(event.AfterRows))
	}
	before := event.BeforeRows[0]
	after := event.AfterRows[0]
	if got := before[0].Value; got != int64(1) {
		t.Fatalf("before[0] = %#v, want int64(1)", got)
	}
	if !before[1].Skipped {
		t.Fatal("before[1] was not marked skipped")
	}
	if got := after[0].Value; got != int64(2) {
		t.Fatalf("after[0] = %#v, want int64(2)", got)
	}
	if !after[1].Null {
		t.Fatal("after[1] was not marked NULL")
	}
}

func TestColumnValueHelpers(t *testing.T) {
	t.Parallel()

	table := &TableMapEvent{
		TableID:     1,
		ColumnCount: 4,
		ColumnTypeDef: []byte{
			common.MySQLTypeNewDecimal,
			common.MySQLTypeBit,
			common.MySQLTypeJSON,
			common.MySQLTypeGeometry,
		},
		ColumnMetaDef: []uint16{
			4<<8 | 2, // DECIMAL(4,2)
			1<<8 | 1, // BIT(9)
			1,        // 1-byte length
			1,        // 1-byte length
		},
	}
	tables := map[uint64]*TableMapEvent{1: table}

	data := []byte{
		1, 0, 0, 0, 0, 0,
		0, 0,
		2, 0,
		4,
		0x0f,
		0x00,
		0x8c, 0x22, // DECIMAL(4,2) 12.34
		0x01, 0x02, // BIT(9)
		2, 0x03, 0x04, // JSON binary raw payload
		5, 1, 0, 0, 0, 0x01, // SRID=1, WKB payload
	}

	event, err := decodeRowsEvent(data, testFmtDescEvent(), tables, common.WriteRowsEventV2)
	if err != nil {
		t.Fatal(err)
	}
	row := event.Rows[0]
	if got, ok := row[0].DecimalString(); !ok || got != "12.34" {
		t.Fatalf("decimal = %q/%v, want 12.34/true", got, ok)
	}
	if got, ok := row[1].Bit(); !ok || got != 0x0102 {
		t.Fatalf("bit = %d/%v, want 258/true", got, ok)
	}
	if got, ok := row[2].JSONBinary(); !ok || string(got) != "\x03\x04" {
		t.Fatalf("json raw = %v/%v, want [3 4]/true", got, ok)
	}
	srid, wkb, ok := row[3].Geometry()
	if !ok || srid != 1 || string(wkb) != "\x01" {
		t.Fatalf("geometry = %d %v %v, want 1 [1] true", srid, wkb, ok)
	}
}

func TestRowsEventDecodesMoreColumnTypes(t *testing.T) {
	t.Parallel()

	table := &TableMapEvent{
		TableID:     1,
		ColumnCount: 19,
		ColumnTypeDef: []byte{
			common.MySQLTypeTiny,
			common.MySQLTypeShort,
			common.MySQLTypeLonglong,
			common.MySQLTypeInt24,
			common.MySQLTypeYear,
			common.MySQLTypeFloat,
			common.MySQLTypeDouble,
			common.MySQLTypeTimestamp,
			common.MySQLTypeDate,
			common.MySQLTypeDatetime,
			common.MySQLTypeTime,
			common.MySQLTypeTimestamp2,
			common.MySQLTypeDatetime2,
			common.MySQLTypeTime2,
			common.MySQLTypeEnum,
			common.MySQLTypeSet,
			common.MySQLTypeBlob,
			common.MySQLTypeTinyBlob,
			common.MySQLTypeLongBlob,
		},
		ColumnMetaDef: []uint16{
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			2, 2, 2,
			1, 1,
			2, 0, 0,
		},
	}

	data := []byte{
		1, 0, 0, 0, 0, 0,
		0, 0,
		2, 0,
		19,
		0xff, 0xff, 0x07,
		0x00, 0x00, 0x00,
		0xfe,
		0xfe, 0xff,
		9, 0, 0, 0, 0, 0, 0, 0,
		0xfe, 0xff, 0xff,
		125,
	}
	data = binary.LittleEndian.AppendUint32(data, math.Float32bits(1.5))
	data = binary.LittleEndian.AppendUint64(data, math.Float64bits(2.5))
	data = binary.LittleEndian.AppendUint32(data, 1)
	date := uint32(2024*16*32 + 6*32 + 1)
	data = append(data, byte(date), byte(date>>8), byte(date>>16))
	data = binary.LittleEndian.AppendUint64(data, 20240601123456)
	timeVal := uint32(123456)
	data = append(data, byte(timeVal), byte(timeVal>>8), byte(timeVal>>16))
	data = append(data, 1, 2, 3, 4, 5)
	data = append(data, 1, 2, 3, 4, 5, 6)
	data = append(data, 1, 2, 3, 4)
	data = append(data, 3)
	data = append(data, 0x05)
	data = append(data, 2, 0, 'o', 'k')
	data = append(data, 1, 'x')
	data = append(data, 1, 0, 0, 0, 'z')

	event, err := decodeRowsEvent(data, testFmtDescEvent(), map[uint64]*TableMapEvent{1: table}, common.WriteRowsEventV2)
	if err != nil {
		t.Fatal(err)
	}
	row := event.Rows[0]
	if row[0].Value != int64(-2) || row[1].Value != int64(-2) || row[2].Value != int64(9) || row[3].Value != int64(-2) {
		t.Fatalf("integer values = %#v", row[:4])
	}
	if row[4].Value != 2025 || row[7].Value.(time.Time).Unix() != 1 {
		t.Fatalf("year/timestamp = %#v/%#v", row[4], row[7])
	}
	if string(row[16].Raw) != "ok" || string(row[17].Raw) != "x" || string(row[18].Raw) != "z" {
		t.Fatalf("blob values = %q/%q/%q", row[16].Raw, row[17].Raw, row[18].Raw)
	}
	if got, ok := row[14].Value.(uint64); !ok || got != 3 {
		t.Fatalf("enum = %#v", row[14].Value)
	}
}
