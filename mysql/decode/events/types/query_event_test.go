package types

import (
	"encoding/binary"
	"testing"

	"github.com/Infranite/go-dblog/mysql/common"
)

func TestQueryEventDecodesStatusVars(t *testing.T) {
	t.Parallel()

	status := []byte{
		common.QFlags2Code, 0x34, 0x12, 0, 0,
		common.QSQLModeCode, 1, 0, 0, 0, 0, 0, 0, 0,
		common.QAutoIncrement, 2, 0, 3, 0,
		common.QCharsetCode, 33, 0, 45, 0, 46, 0,
		common.QTimeZoneCode, 3, 'U', 'T', 'C',
		common.QMicroseconds, 0x40, 0x0d, 0x03,
	}

	data := make([]byte, 0, 13+len(status)+len("db")+1+len("update t set c=1"))
	data = binary.LittleEndian.AppendUint32(data, 10)
	data = binary.LittleEndian.AppendUint32(data, 2)
	data = append(data, 2)
	data = binary.LittleEndian.AppendUint16(data, 0)
	data = binary.LittleEndian.AppendUint16(data, uint16(len(status)))
	data = append(data, status...)
	data = append(data, 'd', 'b', 0)
	data = append(data, "update t set c=1"...)

	body, err := new(QueryEvent).Decode(
		WithData(data),
		WithContext(&EventContext{Description: &FmtDescEvent{BinlogVersion: 4}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	event := body.(*QueryEvent)
	if event.Status.Flags2 != 0x1234 || event.Status.SQLMode != 1 {
		t.Fatalf("status flags/sqlmode = %x/%x", event.Status.Flags2, event.Status.SQLMode)
	}
	if event.Status.AutoIncrementIncrement != 2 || event.Status.AutoIncrementOffset != 3 {
		t.Fatalf("auto increment = %d/%d", event.Status.AutoIncrementIncrement, event.Status.AutoIncrementOffset)
	}
	if event.Status.ClientCharset != 33 || event.Status.CollationConnection != 45 || event.Status.CollationServer != 46 {
		t.Fatalf("charset = %#v", event.Status)
	}
	if event.Status.TimeZone != "UTC" || event.Status.Microseconds != 200000 {
		t.Fatalf("timezone/microseconds = %q/%d", event.Status.TimeZone, event.Status.Microseconds)
	}
	if err := event.Statue(); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeQueryStatusVarsMoreFields(t *testing.T) {
	t.Parallel()

	status, err := DecodeQueryStatusVars([]byte{
		common.QCatalog, 3, 's', 't', 'd', 0,
		common.QCatalogNZCode, 3, 'c', 'a', 't',
		common.QLCTimeNamesCode, 8, 0,
		common.QCharsetDatabaseCode, 45, 0,
		common.QTableMapForUpdateCode, 1, 2, 3, 4, 5, 6, 7, 8,
		common.QMasterDataWrittenCode, 9, 0, 0, 0,
		common.QInvokers, 1, 'u', 1, 'h',
		common.QUpdatedDBNames, 2, 'a', 0, 'b', 0,
		0xee, 1, 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.Catalog != "std" || status.CatalogNZ != "cat" || status.LCTimeNames != 8 || status.CharsetDatabase != 45 {
		t.Fatalf("status = %#v", status)
	}
	if status.TableMapForUpdate != 0x0807060504030201 || status.MasterDataWritten != 9 {
		t.Fatalf("status numeric = %#v", status)
	}
	if status.InvokerUser != "u" || status.InvokerHost != "h" {
		t.Fatalf("invoker = %#v", status)
	}
	if len(status.UpdatedDBNames) != 2 || status.UpdatedDBNames[0] != "a" || status.UpdatedDBNames[1] != "b" {
		t.Fatalf("updated db names = %#v", status.UpdatedDBNames)
	}
	if len(status.UnknownStatusVarsPayload) != 3 {
		t.Fatalf("unknown payload = %v", status.UnknownStatusVarsPayload)
	}
}
