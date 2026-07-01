package redis

import "github.com/Infranite/go-dblog/redis/decode/events/types"

// Command is one decoded Redis command.
type Command = types.Command

// CommandPlugin extends command parsing for product-specific Redis dialects.
type CommandPlugin = types.CommandPlugin

// Event adapts a Redis AOF command to dblog.Event.
type Event = types.Event
