# PostgreSQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/postgres.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/postgres)

This module is the PostgreSQL-family backend for `go-dblog`. It decodes text
logical decoding records, reads live `test_decoding` output through SQL slot
polling or PostgreSQL replication protocol, and exposes transaction and row
changes as typed events.

[中文](./README.zh-CN.md)

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need PostgreSQL-family
logical decoding text parsing.

## Installation

Current release:

```bash
go get github.com/Infranite/go-dblog/postgres@v0.4.0
```

The repository tag for this module is `postgres/v0.4.0`; callers use the
semantic version query above with `go get`.

Requirements:

- Go 1.25 or later.
- Logical decoding text records such as `BEGIN`, `COMMIT`, and
  `table schema.table: INSERT: col[type]:value`.
- For live reading, a PostgreSQL DSN and a `test_decoding` logical slot name.
  Add `replication=database` to the DSN to use wire-level replication protocol.

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres"
)

func main() {
	var registry dblog.Registry
	if err := postgres.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(postgres.Driver,
		dblog.WithSource(dblog.Source{Name: "slot"}),
		dblog.WithReader(strings.NewReader("table public.users: INSERT: id[integer]:1 name[text]:'Ada'\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		change := event.Body().(postgres.Change)
		fmt.Println(event.Kind(), change.Schema, change.Table, len(change.Columns))
	}
}
```

## Documentation

| Topic | English | 中文 |
|---|---|---|
| Features, scope, package structure, live readers, flashback, and plugins | [doc/FEATURES.md](./doc/FEATURES.md) | [doc/FEATURES.zh-CN.md](./doc/FEATURES.zh-CN.md) |
| Offline, live reader, plugin, and flashback examples | [doc/EXAMPLES.md](./doc/EXAMPLES.md) | [doc/EXAMPLES.zh-CN.md](./doc/EXAMPLES.zh-CN.md) |
| Project roadmap and release scope | [../doc/ROADMAP.md](../doc/ROADMAP.md#postgresql-family) | [../doc/ROADMAP.zh-CN.md](../doc/ROADMAP.zh-CN.md#postgresql-族) |
| Development and contribution flow | [../doc/DEVELOPMENT.md](../doc/DEVELOPMENT.md) | [../doc/DEVELOPMENT.zh-CN.md](../doc/DEVELOPMENT.zh-CN.md) |

## License

Apache License 2.0. See [LICENSE](../LICENSE).
