package types

import (
	"strconv"

	"github.com/Infranite/go-dblog"
)

// Event adapts a MongoDB JSON change line to dblog.Event.
type Event struct {
	source   dblog.Source
	position int
	raw      []byte
	change   Change
}

// NewEvent creates a MongoDB event.
func NewEvent(source dblog.Source, position int, raw []byte, change Change) Event {
	if source.Driver == "" {
		source.Driver = Driver
	}
	return Event{
		source:   source,
		position: position,
		raw:      append([]byte(nil), raw...),
		change:   change,
	}
}

func (e Event) SourceDriver() string {
	if e.source.Driver == "" {
		return Driver
	}
	return e.source.Driver
}

func (e Event) SourceName() string { return e.source.Name }

func (e Event) PositionDriver() string { return Driver }

func (e Event) PositionString() string { return strconv.Itoa(e.position) }

func (e Event) Kind() string { return e.change.Operation }

func (e Event) Raw() []byte { return append([]byte(nil), e.raw...) }

func (e Event) Body() any { return e.change }

func (e Event) Reverse() (any, bool) {
	switch e.change.Operation {
	case OperationInsert:
		if len(e.change.DocumentKey) == 0 {
			return nil, false
		}
		return Command{
			Operation:  OperationDelete,
			Database:   e.change.Database,
			Collection: e.change.Collection,
			Filter:     e.change.DocumentKey,
		}, true
	case OperationDelete:
		if len(e.change.Document) == 0 {
			return nil, false
		}
		return Command{
			Operation:  OperationInsert,
			Database:   e.change.Database,
			Collection: e.change.Collection,
			Document:   e.change.Document,
		}, true
	default:
		return nil, false
	}
}

var _ dblog.Event = Event{}
