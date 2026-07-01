// Package backend registers the MySQL-family dblog backend.
package backend

import (
	"errors"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mysql/decode/decoder"
)

// Driver is the MySQL-family backend driver name.
const Driver = "mysql"

var errPathRequired = errors.New("mysql backend path is required")

// Backend opens MySQL-family binlog decoders.
type Backend struct{}

func (Backend) Driver() string { return Driver }

func (Backend) Open(options dblog.OpenOptions) (dblog.Decoder[dblog.Event], error) {
	path := options.Path()
	if path == "" {
		path = options.DSN()
	}
	if path == "" {
		path = options.Source().Name
	}
	if path == "" {
		return nil, errPathRequired
	}

	fileDecoder, err := decoder.NewBinFileDecoder(path)
	if err != nil {
		return nil, err
	}

	source := decoder.Source{Driver: Driver, Name: path}
	if options.Source().Driver != "" {
		source.Driver = options.Source().Driver
	}
	if options.Source().Name != "" {
		source.Name = options.Source().Name
	}
	d := decoder.WrapDblogDecoder(source, fileDecoder)
	return dblog.NewSeqDecoder(dblog.Events(d), d.Close), nil
}

// Register adds Backend to a registry, or to dblog.DefaultRegistry when nil.
func Register(registry *dblog.Registry) error {
	if registry == nil {
		return dblog.Register(Backend{})
	}
	return registry.Register(Backend{})
}
