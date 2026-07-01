package types

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/liipx/go-mysql-binlog/binlog/common"
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

func TestBasicEventDecoders(t *testing.T) {
	t.Parallel()

	startData := make([]byte, 56)
	binary.LittleEndian.PutUint16(startData, 4)
	copy(startData[2:], "8.0.36")
	binary.LittleEndian.PutUint32(startData[52:], 123)
	startBody, err := new(StartEventV3).Decode(WithData(startData))
	if err != nil {
		t.Fatal(err)
	}
	if startBody.(*StartEventV3).MySQLVersion != "8.0.36" {
		t.Fatalf("start = %#v", startBody)
	}

	stopBody, err := new(StopEvent).Decode(WithEventType(common.StopEvent))
	if err != nil {
		t.Fatal(err)
	}
	if stopBody.(*StopEvent).EventType != common.StopEvent {
		t.Fatalf("stop = %#v", stopBody)
	}

	fileBody, err := new(FileEvent).Decode(WithData(append(le32(7), "block"...)), WithEventType(common.AppendBlockEvent))
	if err != nil {
		t.Fatal(err)
	}
	if string(fileBody.(*FileEvent).BlockData) != "block" {
		t.Fatalf("file = %#v", fileBody)
	}

	execBody, err := new(ExecLoadEvent).Decode(WithData(le32(9)))
	if err != nil {
		t.Fatal(err)
	}
	if execBody.(*ExecLoadEvent).FileID != 9 {
		t.Fatalf("exec = %#v", execBody)
	}
}

func TestLoadAndExecuteLoadQueryEvents(t *testing.T) {
	t.Parallel()

	loadData := make([]byte, 0)
	loadData = append(loadData, le32(1)...)
	loadData = append(loadData, le32(2)...)
	loadData = append(loadData, le32(3)...)
	loadData = append(loadData, 3, 2)
	loadData = append(loadData, le32(1)...)
	loadData = append(loadData, ',', '"', '\n', 0, '\\', 0, 0, 1)
	loadData = append(loadData, 'c', 0)
	loadData = append(loadData, 't', 'b', 'l', 0, 'd', 'b', 0, 'f', 'i', 'l', 'e')

	body, err := new(LoadEvent).Decode(WithData(loadData), WithEventType(common.LoadEvent))
	if err != nil {
		t.Fatal(err)
	}
	event := body.(*LoadEvent)
	if event.TableName != "tbl" || event.SchemaName != "db" || event.FileName != "file" || len(event.FieldNames) != 1 {
		t.Fatalf("load = %#v", event)
	}

	execData := make([]byte, 0)
	execData = append(execData, le32(1)...)
	execData = append(execData, le32(2)...)
	execData = append(execData, 2)
	execData = append(execData, 0, 0)
	execData = append(execData, 1, 0)
	execData = append(execData, le32(3)...)
	execData = append(execData, le32(4)...)
	execData = append(execData, le32(5)...)
	execData = append(execData, 1)
	execData = append(execData, 0xff)
	execData = append(execData, 'd', 'b', 0)
	execData = append(execData, "load data"...)

	body, err = new(ExecuteLoadQueryEvent).Decode(WithData(execData))
	if err != nil {
		t.Fatal(err)
	}
	exec := body.(*ExecuteLoadQueryEvent)
	if exec.Schema != "db" || exec.Query != "load data" || exec.FileID != 3 {
		t.Fatalf("execute load = %#v", exec)
	}
}

func TestVariableEvents(t *testing.T) {
	t.Parallel()

	randData := append(le64(1), le64(2)...)
	randBody, err := new(RandEvent).Decode(WithData(randData))
	if err != nil {
		t.Fatal(err)
	}
	if randBody.(*RandEvent).Seed2 != 2 {
		t.Fatalf("rand = %#v", randBody)
	}

	userData := make([]byte, 0)
	userData = append(userData, le32(1)...)
	userData = append(userData, 'v', 0)
	userData = append(userData, 1, le32(33)[0], le32(33)[1], le32(33)[2], le32(33)[3])
	userData = append(userData, le32(3)...)
	userData = append(userData, 'a', 'b', 'c', 0xff)
	userBody, err := new(UserVarEvent).Decode(WithData(userData))
	if err != nil {
		t.Fatal(err)
	}
	user := userBody.(*UserVarEvent)
	if user.Name != "v" || string(user.Value) != "abc" || !user.HasFlags {
		t.Fatalf("user var = %#v", user)
	}

	incidentBody, err := new(IncidentEvent).Decode(WithData([]byte{1, 0, 3, 'b', 'a', 'd'}))
	if err != nil {
		t.Fatal(err)
	}
	if incidentBody.(*IncidentEvent).Message != "bad" {
		t.Fatalf("incident = %#v", incidentBody)
	}

	heartbeatBody, err := new(HeartbeatEvent).Decode(WithData([]byte("mysql-bin.1\x00")), WithEventType(common.HeartbeatEvent))
	if err != nil {
		t.Fatal(err)
	}
	if heartbeatBody.(*HeartbeatEvent).LogIdent != "mysql-bin.1" {
		t.Fatalf("heartbeat = %#v", heartbeatBody)
	}

	rowsQueryBody, err := new(RowsQueryEvent).Decode(WithData([]byte("select 1")))
	if err != nil {
		t.Fatal(err)
	}
	if rowsQueryBody.(*RowsQueryEvent).Query != "select 1" {
		t.Fatalf("rows query = %#v", rowsQueryBody)
	}
}

func TestGTIDEvents(t *testing.T) {
	t.Parallel()

	sid := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	gtidData := append([]byte{1}, sid...)
	gtidData = append(gtidData, le64(10)...)
	gtidData = append(gtidData, logicalTimestampTypeCode)
	gtidData = append(gtidData, le64(2)...)
	gtidData = append(gtidData, le64(3)...)
	gtidData = append(gtidData, []byte{1, 2, 3, 4, 5, 6, 7}...)
	gtidData = append(gtidData, 4)
	gtidData = append(gtidData, le32(80036)...)
	gtidData = append(gtidData, 'x')

	body, err := new(GTIDLogEvent).Decode(WithData(gtidData), WithEventType(common.GTIDEvent))
	if err != nil {
		t.Fatal(err)
	}
	gtid := body.(*GTIDLogEvent)
	if gtid.GNO != 10 || gtid.LastCommitted != 2 || gtid.SequenceNumber != 3 || !strings.Contains(gtid.SIDString(), "00010203") {
		t.Fatalf("gtid = %#v", gtid)
	}

	prevData := make([]byte, 0)
	prevData = append(prevData, le64(1)...)
	prevData = append(prevData, sid...)
	prevData = append(prevData, le64(1)...)
	prevData = append(prevData, le64(7)...)
	prevData = append(prevData, le64(9)...)
	prevBody, err := new(PreviousGTIDsEvent).Decode(WithData(prevData))
	if err != nil {
		t.Fatal(err)
	}
	if got := prevBody.(*PreviousGTIDsEvent).String(); !strings.Contains(got, ":7-8") {
		t.Fatalf("previous gtid string = %s", got)
	}
}

func TestContextAndXAEvents(t *testing.T) {
	t.Parallel()

	txData := append(make([]byte, transactionContextHeaderSize), 'p')
	binary.LittleEndian.PutUint32(txData, 1)
	body, err := new(TransactionContextEvent).Decode(WithData(txData))
	if err != nil {
		t.Fatal(err)
	}
	if body.(*TransactionContextEvent).ThreadID != 1 || string(body.(*TransactionContextEvent).Payload) != "p" {
		t.Fatalf("transaction context = %#v", body)
	}

	viewData := append(make([]byte, viewChangeHeaderSize), 'v')
	body, err = new(ViewChangeEvent).Decode(WithData(viewData))
	if err != nil {
		t.Fatal(err)
	}
	if string(body.(*ViewChangeEvent).Payload) != "v" {
		t.Fatalf("view change = %#v", body)
	}

	xaData := []byte{1}
	xaData = append(xaData, le32(2)...)
	xaData = append(xaData, le32(1)...)
	xaData = append(xaData, le32(1)...)
	xaData = append(xaData, 'g', 'b')
	body, err = new(XAPrepareLogEvent).Decode(WithData(xaData))
	if err != nil {
		t.Fatal(err)
	}
	xa := body.(*XAPrepareLogEvent)
	if !xa.OnePhase || string(xa.GTRID) != "g" || string(xa.BQUAL) != "b" {
		t.Fatalf("xa = %#v", xa)
	}

	payloadBody, err := new(TransactionPayloadEvent).Decode(WithData([]byte("payload")), WithEventType(common.TransactionPayloadEvent))
	if err != nil {
		t.Fatal(err)
	}
	if string(payloadBody.(*TransactionPayloadEvent).Payload) != "payload" {
		t.Fatalf("payload = %#v", payloadBody)
	}
}
