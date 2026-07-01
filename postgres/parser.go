package postgres

import (
	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/decoder"
)

// ParseLine parses one PostgreSQL logical decoding text line.
func ParseLine(source dblog.Source, position int, line string) (Event, error) {
	return decoder.ParseLine(source, position, line)
}
