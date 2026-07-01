package decoder

import (
	"bufio"
	"io"
	"iter"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

// Decoder streams PostgreSQL logical decoding text lines.
type Decoder struct {
	source        dblog.Source
	scanner       *bufio.Scanner
	close         func() error
	plugins       []types.EventPlugin
	startPosition int
}

// NewDecoder creates a decoder over PostgreSQL logical decoding text.
func NewDecoder(source dblog.Source, reader io.Reader, close func() error, opts ...Option) *Decoder {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
		scanner:       scanner,
		close:         close,
		plugins:       cfg.eventPlugins,
		startPosition: cfg.startPosition,
	}
}

func (d *Decoder) Events() iter.Seq2[dblog.Event, error] {
	return func(yield func(dblog.Event, error) bool) {
		if d == nil || d.scanner == nil {
			return
		}
		position := 0
		for d.scanner.Scan() {
			position++
			line := d.scanner.Text()
			if position <= d.startPosition {
				continue
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			event, err := parseLine(d.source, position, line, d.plugins)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(event, nil) {
				return
			}
		}
		if err := d.scanner.Err(); err != nil {
			yield(nil, err)
		}
	}
}

func (d *Decoder) Close() error {
	if d == nil || d.close == nil {
		return nil
	}
	return d.close()
}
