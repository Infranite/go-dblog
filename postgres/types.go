package postgres

import "github.com/Infranite/go-dblog/postgres/decode/events/types"

// Column is one decoded column value from a logical decoding change.
type Column = types.Column

// Change is one decoded row-level logical decoding change.
type Change = types.Change

// Transaction is a BEGIN or COMMIT record.
type Transaction = types.Transaction

// EventPlugin extends PostgreSQL logical decoding text parsing.
type EventPlugin = types.EventPlugin

// Event adapts a PostgreSQL logical decoding line to dblog.Event.
type Event = types.Event
