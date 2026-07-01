package mongo

import (
	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/decoder"
)

// ParseLine parses one MongoDB oplog or change stream JSON line.
func ParseLine(source dblog.Source, position int, line string) (Event, error) {
	return decoder.ParseLine(source, position, line)
}
