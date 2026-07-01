package decoder

import (
	"fmt"
	"iter"

	"github.com/Infranite/go-dblog/mysql/decode/events"
)

const driverMySQL = "mysql"

// Position identifies a location in a MySQL-family binary log.
type Position struct {
	File string
	Pos  int64
}

// Driver returns the backend driver name.
func (p Position) Driver() string {
	return driverMySQL
}

// String returns the display form of the binlog position.
func (p Position) String() string {
	if p.File == "" {
		return fmt.Sprintf("%d", p.Pos)
	}
	return fmt.Sprintf("%s:%d", p.File, p.Pos)
}

// DblogEvent adapts a MySQL-family binlog event to the shared dblog.Event API.
type DblogEvent struct {
	source Source
	event  *events.Event
}

// Source identifies a MySQL-family binlog source.
type Source struct {
	Driver string
	Name   string
}

// SourceDriver returns the source driver name.
func (e DblogEvent) SourceDriver() string {
	if e.source.Driver == "" {
		return driverMySQL
	}
	return e.source.Driver
}

// SourceName returns the source display name.
func (e DblogEvent) SourceName() string {
	return e.source.Name
}

// PositionDriver returns the position driver name.
func (e DblogEvent) PositionDriver() string {
	return driverMySQL
}

// PositionString returns the binlog file and end position for the event.
func (e DblogEvent) PositionString() string {
	return e.position().String()
}

// Kind returns the backend event kind.
func (e DblogEvent) Kind() string {
	if e.event == nil || e.event.Header == nil {
		return ""
	}
	return e.event.Header.Type()
}

// Raw returns the raw event bytes when they are available.
func (e DblogEvent) Raw() []byte {
	if e.event == nil || e.event.Header == nil {
		return nil
	}
	raw := append([]byte{}, e.event.Header.Data...)
	if e.event.Body != nil {
		raw = append(raw, e.event.Body.Encode()...)
	}
	if len(e.event.ChecksumVal) != 0 {
		raw = append(raw, e.event.ChecksumVal...)
	}
	return raw
}

// Body returns the backend-specific typed event body.
func (e DblogEvent) Body() any {
	if e.event == nil {
		return nil
	}
	return e.event.Body
}

// Native returns the original MySQL-family binlog event.
func (e DblogEvent) Native() *events.Event {
	return e.event
}

// DblogDecoder adapts BinFileDecoder to the shared dblog.Decoder API.
type DblogDecoder struct {
	decoder *BinFileDecoder
	source  Source
}

// NewDblogDecoder opens a MySQL-family binlog file as a shared dblog decoder.
func NewDblogDecoder(path string, opts ...BinFileDecodeOptFunc) (*DblogDecoder, error) {
	fileDecoder, err := NewBinFileDecoder(path, opts...)
	if err != nil {
		return nil, err
	}
	return WrapDblogDecoder(Source{Driver: driverMySQL, Name: path}, fileDecoder), nil
}

// WrapDblogDecoder adapts an existing BinFileDecoder to the shared dblog API.
func WrapDblogDecoder(source Source, decoder *BinFileDecoder) *DblogDecoder {
	if source.Driver == "" {
		source.Driver = driverMySQL
	}
	return &DblogDecoder{decoder: decoder, source: source}
}

// Events returns decoded binlog events through the shared dblog API.
func (d *DblogDecoder) Events() iter.Seq2[*DblogEvent, error] {
	return func(yield func(*DblogEvent, error) bool) {
		if d == nil || d.decoder == nil {
			return
		}
		for event, err := range d.decoder.Events() {
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(&DblogEvent{source: d.source, event: event}, nil) {
				return
			}
		}
	}
}

// Close closes the underlying binlog decoder.
func (d *DblogDecoder) Close() error {
	if d == nil || d.decoder == nil {
		return nil
	}
	return d.decoder.Close()
}

func (e DblogEvent) position() Position {
	if e.event == nil || e.event.Header == nil {
		return Position{File: e.source.Name}
	}
	return Position{File: e.source.Name, Pos: e.event.Header.LogPos}
}
