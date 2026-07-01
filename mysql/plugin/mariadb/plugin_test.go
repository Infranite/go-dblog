package mariadb

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"testing"

	"github.com/Infranite/go-dblog/mysql/common"
	"github.com/Infranite/go-dblog/mysql/decode/events/types"
)

func le32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func le64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func TestPluginRegistersMariaDBEvents(t *testing.T) {
	t.Parallel()

	registry := types.DefaultEventRegistry()
	plugin := Plugin()
	if plugin.Name() != "mariadb" || !plugin.Match(&types.FmtDescEvent{MySQLVersion: "10.11.0-MariaDB"}) || plugin.Match(&types.FmtDescEvent{MySQLVersion: "8.0.36"}) {
		t.Fatalf("plugin match/name failed")
	}
	plugin.Register(registry)

	decoder := registry.GetEventBodyDecoder(AnnotateRowsEvent)
	body, err := decoder.Decode(types.WithData([]byte("insert into t values (1)")))
	if err != nil {
		t.Fatal(err)
	}
	event, ok := body.(*AnnotateRowsLogEvent)
	if !ok {
		t.Fatalf("decoded body = %T, want *AnnotateRowsLogEvent", body)
	}
	if event.Query != "insert into t values (1)" {
		t.Fatalf("query = %q", event.Query)
	}
	if registry.EventTypeName(GTIDListEvent) != "MARIADB_GTID_LIST_EVENT" {
		t.Fatalf("event name = %s", registry.EventTypeName(GTIDListEvent))
	}
}

func TestSimpleMariaDBEvents(t *testing.T) {
	t.Parallel()

	checkpointData := append(le32(9), "mysql-bin"...)
	body, err := new(BinlogCheckpointLogEvent).Decode(types.WithData(checkpointData))
	if err != nil {
		t.Fatal(err)
	}
	if body.(*BinlogCheckpointLogEvent).FileName != "mysql-bin" || string(body.Encode()) != string(checkpointData) {
		t.Fatalf("checkpoint = %#v", body)
	}

	gtidData := append(le64(7), le32(8)...)
	gtidData = append(gtidData, gtidFlagGroupCommitID)
	gtidData = append(gtidData, le64(9)...)
	body, err = new(GTIDLogEvent).Decode(types.WithData(gtidData))
	if err != nil {
		t.Fatal(err)
	}
	gtid := body.(*GTIDLogEvent)
	if gtid.Sequence != 7 || gtid.DomainID != 8 || gtid.CommitID != 9 {
		t.Fatalf("gtid = %#v", gtid)
	}

	encryptionData := append([]byte{1}, le32(2)...)
	encryptionData = append(encryptionData, []byte("123456789012")...)
	body, err = new(StartEncryptionLogEvent).Decode(types.WithData(encryptionData))
	if err != nil {
		t.Fatal(err)
	}
	if body.(*StartEncryptionLogEvent).KeyVersion != 2 || string(body.(*StartEncryptionLogEvent).Nonce) != "123456789012" {
		t.Fatalf("encryption = %#v", body)
	}
}

func TestGTIDListLogEventDecode(t *testing.T) {
	t.Parallel()

	data := []byte{
		1, 0, 0, 0,
		2, 0, 0, 0,
		3, 0, 0, 0,
		4, 0, 0, 0, 0, 0, 0, 0,
	}
	body, err := new(GTIDListLogEvent).Decode(types.WithData(data))
	if err != nil {
		t.Fatal(err)
	}
	event := body.(*GTIDListLogEvent)
	if len(event.GTIDs) != 1 || event.GTIDs[0].DomainID != 2 || event.GTIDs[0].ServerID != 3 || event.GTIDs[0].Sequence != 4 {
		t.Fatalf("gtids = %#v", event.GTIDs)
	}
}

func zlibBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testQueryBody() []byte {
	var data []byte
	data = binary.LittleEndian.AppendUint32(data, 1)
	data = binary.LittleEndian.AppendUint32(data, 0)
	data = append(data, 2)
	data = binary.LittleEndian.AppendUint16(data, 0)
	data = binary.LittleEndian.AppendUint16(data, 0)
	data = append(data, 'd', 'b', 0)
	data = append(data, "create table t(id int)"...)
	return data
}

func TestQueryCompressedLogEventDecode(t *testing.T) {
	t.Parallel()

	body, err := new(QueryCompressedLogEvent).Decode(
		types.WithData(zlibBytes(t, testQueryBody())),
		types.WithContext(&types.EventContext{Description: &types.FmtDescEvent{BinlogVersion: 4}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	event := body.(*QueryCompressedLogEvent)
	if event.Query == nil || event.Query.Schema != "db" || event.Query.Query != "create table t(id int)" {
		t.Fatalf("query = %#v", event.Query)
	}
}

func TestRowsCompressedLogEventDecodeRows(t *testing.T) {
	t.Parallel()

	context := types.NewEventContext()
	context.TableInfo[1] = &types.TableMapEvent{
		TableID:       1,
		ColumnCount:   1,
		ColumnTypeDef: []byte{common.MySQLTypeLong},
		ColumnMetaDef: []uint16{0},
	}

	payload := zlibBytes(t, []byte{
		0x00,       // null bitmap
		7, 0, 0, 0, // LONG
	})
	data := []byte{
		1, 0, 0, 0, 0, 0, // table id
		0, 0, // flags
		1,    // column count
		0x01, // columns-present-bitmap
		0x00, // compression header: zlib, no length bytes
	}
	data = append(data, payload...)

	body, err := new(RowsCompressedLogEvent).Decode(
		types.WithData(data),
		types.WithContext(context),
		types.WithEventType(WriteRowsCompressedEvent),
	)
	if err != nil {
		t.Fatal(err)
	}
	event := body.(*RowsCompressedLogEvent)
	if len(event.Rows) != 1 || event.Rows[0][0].Value != int64(7) {
		t.Fatalf("rows = %#v", event.Rows)
	}
	reader, err := zlib.NewReader(bytes.NewReader(event.CompressedPayload))
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if string(event.UncompressedPayload) != "\x00\x07\x00\x00\x00" {
		t.Fatalf("uncompressed payload = %v", event.UncompressedPayload)
	}
}
