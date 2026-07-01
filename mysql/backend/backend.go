// Package backend registers the MySQL-family dblog backend.
package backend

import (
	"errors"
	"fmt"
	"iter"
	"strconv"
	"strings"

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

	startPos, err := startPosition(options)
	if err != nil {
		return nil, err
	}
	var opts []decoder.BinFileDecodeOptFunc
	if startPos > 0 {
		opts = append(opts, decoder.WithStartPos(startPos))
	}
	fileDecoder, err := decoder.NewBinFileDecoder(path, opts...)
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
	events := dblog.Events(d)
	if startPos > 0 {
		events = eventsAfterPosition(events, startPos)
	}
	return dblog.NewSeqDecoder(events, d.Close), nil
}

// Register adds Backend to a registry, or to dblog.DefaultRegistry when nil.
func Register(registry *dblog.Registry) error {
	if registry == nil {
		return dblog.Register(Backend{})
	}
	return registry.Register(Backend{})
}

func startPosition(options dblog.OpenOptions) (int64, error) {
	position := dblog.StartPositionOf(options)
	if position.Value == "" {
		return 0, nil
	}
	if position.Driver != "" && position.Driver != Driver {
		return 0, fmt.Errorf("mysql: checkpoint driver %q does not match %q", position.Driver, Driver)
	}
	value, err := parsePosition(position.Value)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("mysql: invalid checkpoint position %q", position.Value)
	}
	return value, nil
}

func eventsAfterPosition(seq iter.Seq2[dblog.Event, error], startPos int64) iter.Seq2[dblog.Event, error] {
	return func(yield func(dblog.Event, error) bool) {
		for event, err := range seq {
			if err != nil {
				yield(nil, err)
				return
			}
			position, err := parsePosition(event.PositionString())
			if err == nil && position <= startPos {
				continue
			}
			if !yield(event, nil) {
				return
			}
		}
	}
}

func parsePosition(position string) (int64, error) {
	if index := strings.LastIndex(position, ":"); index >= 0 {
		position = position[index+1:]
	}
	return strconv.ParseInt(position, 10, 64)
}
