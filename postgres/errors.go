package postgres

import "github.com/Infranite/go-dblog/postgres/decode/events/types"

var (
	// ErrReaderRequired is returned when no reader, path, or dsn is provided.
	ErrReaderRequired = types.ErrReaderRequired
	// ErrSlotRequired is returned when live logical decoding has no slot name.
	ErrSlotRequired = types.ErrSlotRequired
	// ErrInvalidLine is returned when a logical decoding line is malformed.
	ErrInvalidLine = types.ErrInvalidLine
)
