package types

import (
	"strconv"

	"github.com/Infranite/go-dblog"
)

// Event adapts a PostgreSQL logical decoding line to dblog.Event.
type Event struct {
	source   dblog.Source
	position int
	raw      []byte
	kind     string
	body     any
}

// NewEvent creates a PostgreSQL event.
func NewEvent(source dblog.Source, position int, raw []byte, kind string, body any) Event {
	if source.Driver == "" {
		source.Driver = Driver
	}
	return Event{
		source:   source,
		position: position,
		raw:      append([]byte(nil), raw...),
		kind:     kind,
		body:     body,
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

func (e Event) Kind() string { return e.kind }

func (e Event) Raw() []byte { return append([]byte(nil), e.raw...) }

func (e Event) Body() any { return e.body }

func (e Event) Reverse() (any, bool) {
	change, ok := e.body.(Change)
	if !ok {
		return nil, false
	}
	switch change.Operation {
	case OperationInsert:
		return deleteSQL(change), true
	case OperationUpdate:
		hasImages := len(change.OldKey) > 0 && len(change.NewTuple) > 0
		if !hasImages || !coversColumns(change.OldKey, change.NewTuple) {
			return nil, false
		}
		return updateSQL(change), true
	case OperationDelete:
		return insertSQL(change), true
	default:
		return nil, false
	}
}

var _ dblog.Event = Event{}

func coversColumns(oldKey, newTuple []Column) bool {
	columns := make(map[string]struct{}, len(oldKey))
	for _, column := range oldKey {
		columns[column.Name] = struct{}{}
	}
	for _, column := range newTuple {
		if _, ok := columns[column.Name]; !ok {
			return false
		}
	}
	return true
}
