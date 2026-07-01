package redis

import (
	"io"

	"github.com/Infranite/go-dblog/redis/decode/decoder"
)

// ParseCommand parses one Redis RESP array command.
func ParseCommand(reader io.Reader) (Command, []byte, error) {
	return decoder.ParseCommand(reader)
}
