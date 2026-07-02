# Redis-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/redis.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/redis)

This module is the Redis-family backend for `go-dblog`. It decodes Redis AOF
RESP array commands and can stream live Redis replication commands.

[中文](./README.zh-CN.md)

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need Redis-family AOF RESP
parsing.

## Installation

Current release:

```bash
go get github.com/Infranite/go-dblog/redis@v0.3.0
```

The repository tag for this module is `redis/v0.3.0`; callers use the semantic
version query above with `go get`.

Requirements:

- Go 1.25 or later.
- Redis AOF command frames encoded as RESP arrays, or a Redis server reachable
  by TCP when opening a live replication stream.

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

## Documentation

| Topic | English | 中文 |
|---|---|---|
| Features, scope, package structure, RDB behavior, flashback, and plugins | [doc/FEATURES.md](./doc/FEATURES.md) | [doc/FEATURES.zh-CN.md](./doc/FEATURES.zh-CN.md) |
| AOF, live reader, plugin, and flashback examples | [doc/EXAMPLES.md](./doc/EXAMPLES.md) | [doc/EXAMPLES.zh-CN.md](./doc/EXAMPLES.zh-CN.md) |
| Project roadmap and release scope | [../doc/ROADMAP.md](../doc/ROADMAP.md#redis-family) | [../doc/ROADMAP.zh-CN.md](../doc/ROADMAP.zh-CN.md#redis-族) |
| Development and contribution flow | [../doc/DEVELOPMENT.md](../doc/DEVELOPMENT.md) | [../doc/DEVELOPMENT.zh-CN.md](../doc/DEVELOPMENT.zh-CN.md) |

## License

Apache License 2.0. See [LICENSE](../LICENSE).
