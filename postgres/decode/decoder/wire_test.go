package decoder

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"
)

func TestIsWireReplicationDSN(t *testing.T) {
	for _, dsn := range []string{
		"postgres://postgres@127.0.0.1/postgres?replication=database",
		"host=127.0.0.1 user=postgres replication=database",
		"host=127.0.0.1 user=postgres replication='database'",
	} {
		if !IsWireReplicationDSN(dsn) {
			t.Fatalf("wire DSN not detected: %s", dsn)
		}
	}
	if IsWireReplicationDSN("postgres://postgres@127.0.0.1/postgres?sslmode=disable") {
		t.Fatal("regular DSN detected as wire replication")
	}
}

func TestParseWireData(t *testing.T) {
	line := "table public.users: INSERT: id[integer]:1"
	data := make([]byte, 25+len(line))
	data[0] = 'w'
	binary.BigEndian.PutUint64(data[9:17], 42)
	copy(data[25:], line)

	got, err := parseWireData(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.lsn != 42 || got.line != line || got.reply {
		t.Fatalf("wire data = %#v", got)
	}
}

func TestParseWireKeepalive(t *testing.T) {
	data := make([]byte, 18)
	data[0] = 'k'
	binary.BigEndian.PutUint64(data[1:9], 99)
	data[17] = 1

	got, err := parseWireData(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.lsn != 99 || !got.reply || got.line != "" {
		t.Fatalf("keepalive = %#v", got)
	}
}

func TestParseWireDataRejectsInvalidPayloads(t *testing.T) {
	for _, data := range [][]byte{nil, []byte{'w'}, []byte{'k'}, []byte{'x'}} {
		if _, err := parseWireData(data); err == nil {
			t.Fatalf("parseWireData(%v) succeeded", data)
		}
	}
}

func TestStandbyStatusData(t *testing.T) {
	got := standbyStatusData(7, postgresEpoch.Add(10*time.Microsecond), true)
	if got[0] != 'r' || got[33] != 1 {
		t.Fatalf("standby status flags = %q/%d", got[0], got[33])
	}
	for _, offset := range []int{1, 9, 17} {
		if binary.BigEndian.Uint64(got[offset:offset+8]) != 7 {
			t.Fatalf("lsn at %d = %d", offset, binary.BigEndian.Uint64(got[offset:offset+8]))
		}
	}
	if binary.BigEndian.Uint64(got[25:33]) != 10 {
		t.Fatalf("timestamp = %d", binary.BigEndian.Uint64(got[25:33]))
	}
}

func TestStartReplicationQueryQuotesSlot(t *testing.T) {
	query := startReplicationQuery(`slot"name`)
	if !strings.Contains(query, `"slot""name"`) {
		t.Fatalf("query = %s", query)
	}
}
