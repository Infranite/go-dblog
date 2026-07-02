package decoder

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

type fakeConn struct {
	reader *strings.Reader
	writes bytes.Buffer
	closed bool
}

func newFakeConn(input string) *fakeConn {
	return &fakeConn{reader: strings.NewReader(input)}
}

func (c *fakeConn) Read(p []byte) (int, error)  { return c.reader.Read(p) }
func (c *fakeConn) Write(p []byte) (int, error) { return c.writes.Write(p) }
func (c *fakeConn) Close() error {
	c.closed = true
	return nil
}

func TestLiveDecoderStreamsReplicationCommands(t *testing.T) {
	conn := newFakeConn(strings.Join([]string{
		"+OK\r\n",
		"+OK\r\n",
		"+FULLRESYNC replid 0\r\n",
		"$0\r\n",
		"*2\r\n$3\r\nSET\r\n$3\r\nkey\r\n",
	}, ""))
	decoder := newLiveDecoder(
		context.Background(),
		dblog.Source{Name: "redis://127.0.0.1:6379"},
		conn,
		nil,
	)

	var got []dblog.Event
	for event, err := range decoder.Events() {
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		got = append(got, event)
	}

	if len(got) != 1 {
		t.Fatalf("events = %d, want 1", len(got))
	}
	if got[0].SourceDriver() != types.Driver || got[0].Kind() != "set" {
		t.Fatalf("event = %s/%s", got[0].SourceDriver(), got[0].Kind())
	}
	if !strings.Contains(conn.writes.String(), "PSYNC") {
		t.Fatalf("handshake writes = %q", conn.writes.String())
	}
}

func TestLiveDecoderSkipsDisklessRDB(t *testing.T) {
	conn := newFakeConn(strings.Join([]string{
		"+OK\r\n",
		"+OK\r\n",
		"+FULLRESYNC replid 0\r\n",
		"$EOF:done\r\n",
		"rdb-payload-done",
		"*1\r\n$4\r\nPING\r\n",
	}, ""))
	decoder := newLiveDecoder(context.Background(), dblog.Source{Name: "redis"}, conn, nil)

	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != "ping" {
			t.Fatalf("event kind = %s, want ping", event.Kind())
		}
		return
	}
	t.Fatal("no events")
}

func TestRedisAddressAddsDefaultPort(t *testing.T) {
	address, username, password, err := redisAddress("redis://user:pass@[::1]/0")
	if err != nil {
		t.Fatal(err)
	}
	if address != "[::1]:6379" || username != "user" || password != "pass" {
		t.Fatalf("redis address = %q %q %q", address, username, password)
	}
}
