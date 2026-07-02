package mongo

import "github.com/Infranite/go-dblog/mongo/decode/events/types"

var (
	// ErrReaderRequired is returned when no reader, path, or dsn is provided.
	ErrReaderRequired = types.ErrReaderRequired
	// ErrCollectionRequired is returned when live change streams have no db.collection source name.
	ErrCollectionRequired = types.ErrCollectionRequired
	// ErrInvalidJSON is returned when a JSON change line is malformed.
	ErrInvalidJSON = types.ErrInvalidJSON
	// ErrUnsupportedOperation is returned for unknown MongoDB operation codes.
	ErrUnsupportedOperation = types.ErrUnsupportedOperation
)
