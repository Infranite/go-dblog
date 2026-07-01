# PostgreSQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/postgres.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/postgres)

This module is the PostgreSQL-family backend for `go-dblog`. It decodes text
logical decoding records and exposes transaction and row changes as typed
events.

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need PostgreSQL-family
logical decoding text parsing.

## Installation

After the first `postgres/v0.1.0` tag is published:

```bash
go get github.com/Infranite/go-dblog/postgres
```

Requirements:

- Go 1.23 or later.
- Logical decoding text records such as `BEGIN`, `COMMIT`, and
  `table schema.table: INSERT: col[type]:value`.

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

## Packages

| Package | Purpose |
|---|---|
| `github.com/Infranite/go-dblog/postgres` | Compatibility facade for common imports. |
| `github.com/Infranite/go-dblog/postgres/backend` | Explicit registration with `dblog.Registry`. |
| `github.com/Infranite/go-dblog/postgres/decode/decoder` | Native streaming decoder, line parser, and plugin options. |
| `github.com/Infranite/go-dblog/postgres/decode/events/types` | Native transaction, change, event, and plugin types. |

## Features

- Transaction records: `BEGIN` and `COMMIT`.
- Row changes in PostgreSQL logical decoding text form.
- Scalar parsing for `null`, booleans, integers, floats, and quoted strings.
- Streaming line decoder with bounded scanner buffers.
- Root registry integration through `postgres/backend`.
- Checkpoint resume through `dblog.WithCheckpoint` when opened through the root
  registry.
- SQL flashbacks for inserts and deletes.
- Event plugins for PostgreSQL-compatible sources with extra line types.

The backend driver name is `pg`, while the module path remains `postgres`.

## Supported Inputs

| Input | Status | CI evidence |
|---|---|---|
| `BEGIN` and `COMMIT` records from logical decoding text output | Supported | Unit tests and PostgreSQL fixture job generated from `postgres:16`. |
| Row changes in `table schema.table: OPERATION: col[type]:value` form | Supported | Unit tests, fixture job, and `FuzzParseLine` smoke target. |
| Empty table or operation names | Rejected | Parser tests and fuzz smoke target. |
| Logical replication protocol messages | Planned | Not part of the offline parser release line. |

## Flashback Scope

| Event | Flashback output |
|---|---|
| `insert` | `DELETE FROM ... WHERE ...;` |
| `delete` | `INSERT INTO ... VALUES ...;` |
| `update`, `begin`, `commit` | No flashback output. |

Update flashback needs old and new row images. The text format this parser
accepts does not guarantee both, so the backend does not synthesize unsafe SQL.

## Event Plugins

Use `decoder.WithEventPlugins` to handle line families outside the built-in
logical decoding text records.

```go
package main

import (
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/decoder"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

type messagePlugin struct{}

func (messagePlugin) Name() string { return "message" }
func (messagePlugin) Match(line string) bool {
	return strings.HasPrefix(line, "message ")
}
func (messagePlugin) Decode(source dblog.Source, position int, line string) (types.Event, error) {
	return types.NewEvent(source, position, []byte(line), "message", line), nil
}

func main() {
	_ = decoder.NewDecoder(
		dblog.Source{Name: "slot"},
		strings.NewReader("message hello\n"),
		nil,
		decoder.WithEventPlugins(messagePlugin{}),
	)
}
```

## Development

From the repository root, run:

```bash
cd postgres && GOWORK=off go test ./...
```

Run the PostgreSQL fixture-backed integration test locally when Docker is
available:

```bash
make integration-postgres
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
