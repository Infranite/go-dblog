// Package dblog defines the shared user-facing contracts for database log
// backends.
package dblog

import "iter"

// Source identifies the database log producer behind an event stream.
type Source struct {
	Driver string
	Name   string
}

// Position identifies a backend-specific location in a database log stream.
type Position struct {
	Driver string
	Value  string
}

// Event is the minimal common shape shared by database log backends.
type Event interface {
	SourceDriver() string
	SourceName() string
	PositionDriver() string
	PositionString() string
	Kind() string
	Raw() []byte
	Body() any
}

// Decoder streams events from one database log source.
type Decoder[T Event] interface {
	Events() iter.Seq2[T, error]
	Close() error
}

// SourceOf returns the common source value for an event.
func SourceOf(event Event) Source {
	if event == nil {
		return Source{}
	}
	return Source{Driver: event.SourceDriver(), Name: event.SourceName()}
}

// PositionOf returns the common position value for an event.
func PositionOf(event Event) Position {
	if event == nil {
		return Position{}
	}
	return Position{Driver: event.PositionDriver(), Value: event.PositionString()}
}

// Bodies filters an event iterator by decoded body type.
func Bodies[T any](seq iter.Seq2[Event, error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		for event, err := range seq {
			if err != nil {
				yield(zero, err)
				return
			}
			if event == nil {
				continue
			}
			body, ok := event.Body().(T)
			if !ok {
				continue
			}
			if !yield(body, nil) {
				return
			}
		}
	}
}

// Events adapts a typed backend decoder to the common event interface.
func Events[T Event](decoder Decoder[T]) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		if decoder == nil {
			return
		}
		for event, err := range decoder.Events() {
			if !yield(event, err) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}
