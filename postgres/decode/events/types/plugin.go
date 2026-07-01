package types

import "github.com/Infranite/go-dblog"

// EventPlugin extends PostgreSQL logical decoding text parsing.
type EventPlugin interface {
	Name() string
	Match(string) bool
	Decode(dblog.Source, int, string) (Event, error)
}
