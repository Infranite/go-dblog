package redis

import "github.com/Infranite/go-dblog/redis/decode/events/types"

var (
	// ErrReaderRequired is returned when no reader, path, or dsn is provided.
	ErrReaderRequired = types.ErrReaderRequired
	// ErrInvalidRESP is returned when a RESP command frame is malformed.
	ErrInvalidRESP = types.ErrInvalidRESP
)
