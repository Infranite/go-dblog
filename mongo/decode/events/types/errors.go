package types

import "errors"

var (
	// ErrReaderRequired is returned when no reader, path, or dsn is provided.
	ErrReaderRequired = errors.New("mongo: reader or path is required")
	// ErrCollectionRequired is returned when live change streams have no db.collection source name.
	ErrCollectionRequired = errors.New("mongo: db.collection source name is required")
	// ErrInvalidJSON is returned when a JSON change line is malformed.
	ErrInvalidJSON = errors.New("mongo: invalid json")
	// ErrUnsupportedOperation is returned for unknown MongoDB operation codes.
	ErrUnsupportedOperation = errors.New("mongo: unsupported operation")
)
