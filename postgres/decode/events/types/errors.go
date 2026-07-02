package types

import "errors"

var (
	// ErrReaderRequired is returned when no reader, path, or dsn is provided.
	ErrReaderRequired = errors.New("postgres: reader or path is required")
	// ErrSlotRequired is returned when live logical decoding has no slot name.
	ErrSlotRequired = errors.New("postgres: logical replication slot is required")
	// ErrInvalidLine is returned when a logical decoding line is malformed.
	ErrInvalidLine = errors.New("postgres: invalid logical decoding line")
)
