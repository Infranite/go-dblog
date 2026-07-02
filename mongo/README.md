# MongoDB-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/mongo.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/mongo)

This module is the MongoDB-family backend for `go-dblog`. It decodes newline
delimited JSON records from MongoDB oplog exports or change stream captures,
streams live collection change events from a replica set, and keeps
MongoDB-specific fields in typed events.

[中文](./README.zh-CN.md)

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need MongoDB-family log
parsing.

## Installation

Current release:

```bash
go get github.com/Infranite/go-dblog/mongo@v0.2.0
```

The repository tag for this module is `mongo/v0.2.0`; callers use the semantic
version query above with `go get`.

Requirements:

- Go 1.25 or later.
- One JSON record per line from an oplog export or change stream capture.
- A MongoDB replica set when opening a live change stream through a DSN.

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo"
)

func main() {
	var registry dblog.Registry
	if err := mongo.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(mongo.Driver,
		dblog.WithReader(strings.NewReader(`{"op":"i","ns":"app.users","o":{"_id":1,"name":"Ada"}}`+"\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		change := event.Body().(mongo.Change)
		fmt.Println(event.Kind(), change.Database, change.Collection)
	}
}
```

## Documentation

| Topic | English | 中文 |
|---|---|---|
| Features, scope, package structure, live streams, flashback, and plugins | [doc/FEATURES.md](./doc/FEATURES.md) | [doc/FEATURES.zh-CN.md](./doc/FEATURES.zh-CN.md) |
| Oplog, change stream, live reader, plugin, and flashback examples | [doc/EXAMPLES.md](./doc/EXAMPLES.md) | [doc/EXAMPLES.zh-CN.md](./doc/EXAMPLES.zh-CN.md) |
| Project roadmap and release scope | [../doc/ROADMAP.md](../doc/ROADMAP.md#mongodb-family) | [../doc/ROADMAP.zh-CN.md](../doc/ROADMAP.zh-CN.md#mongodb-族) |
| Development and contribution flow | [../doc/DEVELOPMENT.md](../doc/DEVELOPMENT.md) | [../doc/DEVELOPMENT.zh-CN.md](../doc/DEVELOPMENT.zh-CN.md) |

## License

Apache License 2.0. See [LICENSE](../LICENSE).
