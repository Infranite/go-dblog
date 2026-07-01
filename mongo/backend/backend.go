package backend

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/decoder"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
)

// Driver is the MongoDB-family backend driver name.
const Driver = types.Driver

// Backend opens MongoDB JSON change decoders.
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
	startPosition, err := startPosition(options)
	if err != nil {
		return nil, err
	}
	return decoder.NewDecoder(source, reader, close, decoder.WithStartPosition(startPosition)), nil
}

// Register adds Backend to a registry, or to dblog.DefaultRegistry when nil.
func Register(registry *dblog.Registry) error {
	if registry == nil {
		return dblog.Register(Backend{})
	}
	return registry.Register(Backend{})
}

func startPosition(options dblog.OpenOptions) (int, error) {
	position := dblog.StartPositionOf(options)
	if position.Value == "" {
		return 0, nil
	}
	if position.Driver != "" && position.Driver != Driver {
		return 0, fmt.Errorf("mongo: checkpoint driver %q does not match %q", position.Driver, Driver)
	}
	value, err := strconv.Atoi(position.Value)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("mongo: invalid checkpoint position %q", position.Value)
	}
	return value, nil
}
