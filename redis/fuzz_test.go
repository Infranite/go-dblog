package redis

import (
	"strings"
	"testing"
)

func FuzzParseCommand(f *testing.F) {
	f.Add("*3\r\n$4\r\nSADD\r\n$4\r\ntags\r\n$2\r\ngo\r\n")
	f.Add("*4\r\n$4\r\nHSET\r\n$6\r\nuser:1\r\n$4\r\nname\r\n$3\r\nAda\r\n")
	f.Add("*1\n$0\n\r\n")
	f.Add("*10000000000000\r\n")
	f.Add("SET key value\r\n")

	f.Fuzz(func(t *testing.T, input string) {
		command, raw, err := ParseCommand(strings.NewReader(input))
		if err != nil {
			return
		}
		if command.Name == "" {
			t.Fatal("command name is empty")
		}
		if len(raw) == 0 {
			t.Fatal("raw command is empty")
		}
		_, _ = command.Reverse()
	})
}
