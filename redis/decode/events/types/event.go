package types

import (
	"strconv"

	"github.com/Infranite/go-dblog"
)

// Event adapts a Redis AOF command to dblog.Event.
type Event struct {
	source   dblog.Source
	position int
	raw      []byte
	command  Command
}

// NewEvent creates a Redis event.
func NewEvent(source dblog.Source, position int, raw []byte, command Command) Event {
	if source.Driver == "" {
		source.Driver = Driver
	}
	return Event{
		source:   source,
		position: position,
		raw:      append([]byte(nil), raw...),
		command:  command,
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

func (e Event) Kind() string { return e.command.Name }

func (e Event) Raw() []byte { return append([]byte(nil), e.raw...) }

func (e Event) Body() any { return e.command }

func (e Event) Reverse() (any, bool) {
	command, ok := e.command.Reverse()
	return command, ok
}

var _ dblog.Event = Event{}
