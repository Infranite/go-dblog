package backend

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/decoder"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

// Driver is the Redis-family backend driver name.
const Driver = types.Driver

// Backend opens Redis AOF RESP decoders.
type Backend struct{}

func (Backend) Driver() string { return Driver }

func (Backend) Open(options dblog.OpenOptions) (dblog.Decoder[dblog.Event], error) {
	reader := options.Reader()
	source := options.Source()
	var close func() error
	start, err := startPosition(options)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		path := options.Path()
		if path == "" && isRedisDSN(options.DSN()) {
			return decoder.NewLiveDecoder(
				dblog.ContextOf(options),
				source,
				options.DSN(),
				decoder.WithStartPosition(start),
			)
		}
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
	return decoder.NewDecoder(source, reader, close, decoder.WithStartPosition(start)), nil
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
		return 0, fmt.Errorf("redis: checkpoint driver %q does not match %q", position.Driver, Driver)
	}
	value, err := strconv.Atoi(position.Value)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("redis: invalid checkpoint position %q", position.Value)
	}
	return value, nil
}

func isRedisDSN(dsn string) bool {
	return strings.HasPrefix(strings.TrimSpace(dsn), "redis://")
}
