package events

import (
	"encoding/binary"
	"hash/crc32"
	"strings"
	"testing"

	"github.com/Infranite/go-dblog/mysql/common"
)

func TestDecodeEventHeader(t *testing.T) {
	t.Parallel()

	data := make([]byte, common.DefaultEventHeaderSize)
	binary.LittleEndian.PutUint32(data, 10)
	data[4] = common.QueryEvent
	binary.LittleEndian.PutUint32(data[5:], 11)
	binary.LittleEndian.PutUint32(data[9:], 30)
	binary.LittleEndian.PutUint32(data[13:], 99)
	binary.LittleEndian.PutUint16(data[17:], 3)

	header, err := DecodeEventHeader(data, common.DefaultEventHeaderSize)
	if err != nil {
		t.Fatal(err)
	}
	if header.Type() != "QUERY_EVENT" || header.Timestamp != 10 || header.ServerID != 11 || header.EventSize != 30 || header.LogPos != 99 || header.Flag != 3 {
		t.Fatalf("header = %#v", header)
	}
	if !strings.Contains(header.String(), "QUERY_EVENT") {
		t.Fatalf("String = %s", header.String())
	}
	if _, err := DecodeEventHeader(data[:3], common.DefaultEventHeaderSize); err == nil {
		t.Fatal("short header got nil error")
	}
}

func TestEventValidateData(t *testing.T) {
	t.Parallel()

	header := &EventHeader{
		Data:      make([]byte, common.DefaultEventHeaderSize),
		EventSize: common.DefaultEventHeaderSize + 1,
	}
	event := &Event{Header: header}
	if _, err := event.ValidateData([]byte{1}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := event.ValidateData([]byte{1, 2}, false); err == nil {
		t.Fatal("invalid size got nil error")
	}

	body := []byte{1, 2, 3, common.BinlogChecksumAlgCRC32}
	checksumData := append(append([]byte{}, header.Data...), body...)
	sum := make([]byte, common.BinlogChecksumLength)
	binary.LittleEndian.PutUint32(sum, crc32.ChecksumIEEE(checksumData))
	withChecksum := append(body, sum...)
	header.EventSize = int64(len(header.Data) + len(withChecksum))
	got, err := event.ValidateData(withChecksum, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(body) {
		t.Fatalf("validated body length = %d, want %d", len(got), len(body))
	}
	if _, err := (&Event{}).ValidateData(nil, false); err == nil {
		t.Fatal("nil header got nil error")
	}
}

func TestNewEvent(t *testing.T) {
	t.Parallel()

	if NewEvent() == nil {
		t.Fatal("NewEvent returned nil")
	}
}
