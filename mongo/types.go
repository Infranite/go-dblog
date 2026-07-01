package mongo

import "github.com/Infranite/go-dblog/mongo/decode/events/types"

// Change is one decoded MongoDB oplog or change stream event.
type Change = types.Change

// Command is a MongoDB operation emitted for flashback output.
type Command = types.Command

// EventPlugin extends MongoDB change decoding for product-specific events.
type EventPlugin = types.EventPlugin

// Event adapts a MongoDB JSON change line to dblog.Event.
type Event = types.Event
