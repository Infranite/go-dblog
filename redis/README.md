# Redis-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/redis.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/redis)

This module is the Redis-family backend for `go-dblog`. It decodes Redis AOF
RESP array commands and exposes each command as a typed event.

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need Redis-family AOF RESP
parsing.

## Installation

```bash
go get github.com/Infranite/go-dblog/redis
```

Requirements:

- Go 1.23 or later.
- Redis AOF command frames encoded as RESP arrays.

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis"
)

func main() {
	var registry dblog.Registry
	if err := redis.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(redis.Driver,
		dblog.WithSource(dblog.Source{Name: "appendonly.aof"}),
		dblog.WithReader(strings.NewReader("*3\r\n$4\r\nSADD\r\n$4\r\ntags\r\n$2\r\ngo\r\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		command := event.Body().(redis.Command)
		fmt.Println(event.Kind(), command.Args)
	}
}
```

## Packages

| Package | Purpose |
|---|---|
| `github.com/Infranite/go-dblog/redis` | Compatibility facade for common imports. |
| `github.com/Infranite/go-dblog/redis/backend` | Explicit registration with `dblog.Registry`. |
| `github.com/Infranite/go-dblog/redis/decode/decoder` | Native streaming decoder, RESP parser, and plugin options. |
| `github.com/Infranite/go-dblog/redis/decode/events/types` | Native command, event, and plugin types. |

## Features

- RESP array command parsing for Redis AOF records.
- Lowercase normalized command names.
- Streaming RESP decoder.
- Root registry integration through `redis/backend`.
- Flashback commands for operations that can be safely reversed without reading
  Redis state.
- Command plugins for Redis-compatible products and module commands.

## Flashback Scope

| Command | Flashback output |
|---|---|
| `HSET key field value ...` | `HDEL key field ...` |
| `SADD key member ...` | `SREM key member ...` |
| `LPUSH key value ...` | `LPOP key count` |
| `RPUSH key value ...` | `RPOP key count` |
| `INCR`, `DECR`, `INCRBY`, `DECRBY` | Opposite increment command |

Commands that require previous Redis state, TTLs, or overwritten values do not
emit flashback output.

## Command Plugins

Use `decoder.WithCommandPlugins` to normalize Redis module commands or
Redis-compatible dialects before events are emitted.

```go
package main

import (
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/decoder"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

type renamePlugin struct{}

func (renamePlugin) Name() string { return "rename" }
func (renamePlugin) Match(command types.Command) bool {
	return command.Name == "json.set"
}
func (renamePlugin) Apply(command *types.Command) error {
	command.Name = "jsonset"
	return nil
}

func main() {
	_ = decoder.NewDecoder(
		dblog.Source{Name: "appendonly.aof"},
		strings.NewReader("*2\r\n$8\r\nJSON.SET\r\n$5\r\nkey:1\r\n"),
		nil,
		decoder.WithCommandPlugins(renamePlugin{}),
	)
}
```

## Development

From the repository root, run:

```bash
cd redis && GOWORK=off go test ./...
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
