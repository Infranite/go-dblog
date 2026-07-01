package common

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"testing"
)

type slowReader struct {
	data []byte
}

func (r *slowReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	p[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}

func TestCommonHelpers(t *testing.T) {
	t.Parallel()

	data, err := ReadNBytes(&slowReader{data: []byte{1, 2, 3}}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{1, 2, 3}) {
		t.Fatalf("ReadNBytes = %v", data)
	}

	if got := FixedLengthInt([]byte{1, 2, 3}); got != 0x030201 {
		t.Fatalf("FixedLengthInt = %x", got)
	}
	if got := BitmapByteSize(9); got != 2 {
		t.Fatalf("BitmapByteSize = %d", got)
	}
	if got := EventTypeName(QueryEvent); got != "QUERY_EVENT" {
		t.Fatalf("EventTypeName = %s", got)
	}
	if got := EventTypeName(250); got != "UNKNOWN_EVENT_TYPE" {
		t.Fatalf("unknown EventTypeName = %s", got)
	}
}

func TestLengthEncodedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		num  uint64
		n    int
	}{
		{name: "one byte", data: []byte{250}, num: 250, n: 1},
		{name: "two bytes", data: []byte{0xfc, 1, 2}, num: 0x0201, n: 3},
		{name: "three bytes", data: []byte{0xfd, 1, 2, 3}, num: 0x030201, n: 4},
		{name: "eight bytes", data: []byte{0xfe, 1, 2, 3, 4, 5, 6, 7, 8}, num: 0x0807060504030201, n: 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isNull, n := LengthEncodedInt(tt.data)
			if isNull || got != tt.num || n != tt.n {
				t.Fatalf("LengthEncodedInt = %d/%v/%d, want %d/false/%d", got, isNull, n, tt.num, tt.n)
			}
		})
	}

	if _, isNull, n := LengthEncodedInt([]byte{0xfb}); !isNull || n != 1 {
		t.Fatalf("NULL LengthEncodedInt = %v/%d", isNull, n)
	}
	s, _, n, err := LengthEnodedString([]byte{3, 'a', 'b', 'c'})
	if err != nil || string(s) != "abc" || n != 4 {
		t.Fatalf("LengthEnodedString = %q/%d/%v", s, n, err)
	}
	if _, _, _, err := LengthEnodedString([]byte{3, 'a'}); err == nil {
		t.Fatal("LengthEnodedString short data got nil error")
	}
}

func TestChecksum(t *testing.T) {
	t.Parallel()

	if MysqlVersion("5.6.2-log") != 5<<20|6<<10|2 {
		t.Fatalf("MysqlVersion did not parse patch")
	}
	if !HasChecksum("5.6.2-log") || HasChecksum("5.6.1") {
		t.Fatalf("HasChecksum boundary failed")
	}

	payload := []byte("abc")
	sum := make([]byte, BinlogChecksumLength)
	binary.LittleEndian.PutUint32(sum, crc32.ChecksumIEEE(payload))
	if !ChecksumValidate(BinlogChecksumAlgCRC32, sum, payload) {
		t.Fatal("valid CRC32 rejected")
	}
	sum[0]++
	if ChecksumValidate(BinlogChecksumAlgCRC32, sum, payload) {
		t.Fatal("invalid CRC32 accepted")
	}
	if !ChecksumValidate(BinlogChecksumAlgOff, nil, payload) {
		t.Fatal("checksum off rejected")
	}
}
