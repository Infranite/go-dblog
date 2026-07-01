package types

import "errors"

var (
	// ErrReaderRequired is returned when no reader, path, or dsn is provided.
	ErrReaderRequired = errors.New("redis: reader or path is required")
	// ErrInvalidRESP is returned when a RESP command frame is malformed.
	ErrInvalidRESP = errors.New("redis: invalid resp")
)
