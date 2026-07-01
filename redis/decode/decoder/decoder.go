package decoder

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

// Decoder streams Redis AOF RESP commands.
type Decoder struct {
	source        dblog.Source
	reader        *bufio.Reader
	close         func() error
	plugins       []types.CommandPlugin
	startPosition int
}

// NewDecoder creates a decoder over Redis AOF RESP commands.
func NewDecoder(source dblog.Source, reader io.Reader, close func() error, opts ...Option) *Decoder {
	if source.Driver == "" {
		source.Driver = types.Driver
	}
	cfg := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &Decoder{
		source:        source,
		reader:        bufio.NewReader(reader),
		close:         close,
		plugins:       cfg.commandPlugins,
		startPosition: cfg.startPosition,
	}
}

func (d *Decoder) Events() iter.Seq2[dblog.Event, error] {
	return func(yield func(dblog.Event, error) bool) {
		if d == nil || d.reader == nil {
			return
		}
		position := 0
		for {
			command, raw, err := parseCommand(d.reader)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					yield(nil, err)
				}
				return
			}
			position++
			if position <= d.startPosition {
				continue
			}
			if err := applyCommandPlugins(&command, d.plugins); err != nil {
				yield(nil, err)
				return
			}
			event := types.NewEvent(d.source, position, raw, command)
			if !yield(event, nil) {
				return
			}
		}
	}
}

func (d *Decoder) Close() error {
	if d == nil || d.close == nil {
		return nil
	}
	return d.close()
}

func applyCommandPlugins(command *types.Command, plugins []types.CommandPlugin) error {
	for _, plugin := range plugins {
		if plugin == nil || !plugin.Match(*command) {
			continue
		}
		if err := plugin.Apply(command); err != nil {
			return fmt.Errorf("apply redis command plugin %s: %w", plugin.Name(), err)
		}
	}
	return nil
}
