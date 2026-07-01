package backend

import (
	"os"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/decoder"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

// Driver is the PostgreSQL-family backend driver name.
const Driver = types.Driver

// Backend opens PostgreSQL logical decoding text decoders.
type Backend struct{}

func (Backend) Driver() string { return Driver }

func (Backend) Open(options dblog.OpenOptions) (dblog.Decoder[dblog.Event], error) {
	reader := options.Reader()
	source := options.Source()
	var close func() error
	if reader == nil {
		path := options.Path()
		if path == "" {
			path = options.DSN()
		}
		if path == "" {
			return nil, types.ErrReaderRequired
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		reader = file
		close = file.Close
		if source.Name == "" {
			source.Name = path
		}
	}
	if source.Driver == "" {
		source.Driver = Driver
	}
	return decoder.NewDecoder(source, reader, close), nil
}

// Register adds Backend to a registry, or to dblog.DefaultRegistry when nil.
func Register(registry *dblog.Registry) error {
	if registry == nil {
		return dblog.Register(Backend{})
	}
	return registry.Register(Backend{})
}
