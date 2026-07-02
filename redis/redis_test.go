package redis

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Infranite/go-dblog"
)

var (
	benchmarkRedisCommand Command
	benchmarkRedisRaw     []byte
)

func TestParseCommandParsesRESPArray(t *testing.T) {
	command, raw, err := ParseCommand(strings.NewReader("*3\r\n$4\r\nHSET\r\n$6\r\nuser:1\r\n$4\r\nname\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if command.Name != CommandHSet || !reflect.DeepEqual(command.Args, []string{"user:1", "name"}) {
		t.Fatalf("command = %#v", command)
	}
	if len(raw) == 0 {
		t.Fatal("raw command is empty")
	}
}

func BenchmarkParseCommand(b *testing.B) {
	input := "*3\r\n$4\r\nHSET\r\n$6\r\nuser:1\r\n$4\r\nname\r\n"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		command, raw, err := ParseCommand(strings.NewReader(input))
		if err != nil {
			b.Fatal(err)
		}
		benchmarkRedisCommand = command
		benchmarkRedisRaw = raw
	}
}

func TestBackendStreamsEventsAndFlashbacks(t *testing.T) {
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithSource(dblog.Source{Name: "appendonly.aof"}),
		dblog.WithReader(strings.NewReader(strings.Join([]string{
			"*4\r\n$4\r\nHSET\r\n$6\r\nuser:1\r\n$4\r\nname\r\n$3\r\nAda\r\n",
			"*3\r\n$4\r\nSADD\r\n$4\r\ntags\r\n$2\r\ngo\r\n",
		}, ""))),
	)
	if err != nil {
		t.Fatal(err)
	}
	streamDecoder := decoder
	t.Cleanup(func() {
		if err := streamDecoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	var kinds []string
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		kinds = append(kinds, event.Kind())
	}
	if strings.Join(kinds, ",") != CommandHSet+","+CommandSAdd {
		t.Fatalf("kinds = %v", kinds)
	}

	decoder, err = registry.Open(Driver, dblog.WithReader(strings.NewReader("*2\r\n$4\r\nINCR\r\n$7\r\ncounter\r\n")))
	if err != nil {
		t.Fatal(err)
	}
	flashbackDecoder := decoder
	t.Cleanup(func() {
		if err := flashbackDecoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	var got []any
	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, op)
	}
	want := Command{Name: CommandDecr, Args: []string{"counter"}}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("flashbacks = %#v", got)
	}
}

func TestCommandReverseOmitsStateDependentCommands(t *testing.T) {
	tests := []struct {
		name    string
		command Command
	}{
		{
			name:    "hash set may overwrite an existing field",
			command: Command{Name: CommandHSet, Args: []string{"user:1", "name", "Ada"}},
		},
		{
			name:    "set add may include existing members",
			command: Command{Name: CommandSAdd, Args: []string{"tags", "go"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reverse, ok := tt.command.Reverse()
			if ok {
				t.Fatalf("reverse = %#v, want no flashback", reverse)
			}
		})
	}
}

func TestParseCommandRejectsInvalidRESP(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "inline command", input: "SET key value\r\n"},
		{name: "rdb preamble", input: "REDIS0009\r\n"},
		{name: "mixed rdb and resp stream", input: "REDIS0009\r\n*1\r\n$4\r\nPING\r\n"},
		{name: "empty array", input: "*0\r\n"},
		{name: "oversized array", input: "*8193\r\n"},
		{name: "negative bulk length", input: "*1\r\n$-1\r\n"},
		{name: "oversized bulk length", input: "*1\r\n$8388609\r\n"},
		{name: "bad bulk terminator", input: "*1\r\n$3\r\nSET\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseCommand(strings.NewReader(tt.input))
			if !errors.Is(err, ErrInvalidRESP) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidRESP)
			}
		})
	}
}

func TestBackendRequiresInput(t *testing.T) {
	_, err := Backend{}.Open(nilOptions{})
	if !errors.Is(err, ErrReaderRequired) {
		t.Fatalf("err = %v, want %v", err, ErrReaderRequired)
	}
}
