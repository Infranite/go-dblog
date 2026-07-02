# PostgreSQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/postgres.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/postgres)

This module is the PostgreSQL-family backend for `go-dblog`. It decodes text
logical decoding records, can read live `test_decoding` output through SQL slot
polling or PostgreSQL replication protocol, and exposes transaction and row
changes as typed events.

[中文](./README.zh-CN.md)

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need PostgreSQL-family
logical decoding text parsing.

## Installation

No public tags have been published yet. After the first `v0.2.0` tag set is
published:

```bash
go get github.com/Infranite/go-dblog/postgres@v0.2.0
```

The repository tag for this module is `postgres/v0.2.0`; callers use the
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
- Live SQL logical slot reader for PostgreSQL `test_decoding` output.
- Wire-level logical replication reader for PostgreSQL `test_decoding` output.
- Root registry integration through `postgres/backend`.
- Checkpoint resume through `dblog.WithCheckpoint` when opened through the root
  registry.
- SQL flashbacks for inserts, deletes, and updates with complete old/new tuple
  data.
- Event plugins for PostgreSQL-compatible sources with extra line types.

The backend driver name is `pg`, while the module path remains `postgres`.

## Supported Inputs

| Input | Status | CI evidence |
|---|---|---|
| `BEGIN` and `COMMIT` records from logical decoding text output | Supported | Unit tests and PostgreSQL fixture job generated from `postgres:16`. |
| Row changes in `table schema.table: OPERATION: col[type]:value` form | Supported | Unit tests, fixture job, and `FuzzParseLine` smoke target. |
| `UPDATE: old-key: ... new-tuple: ...` records with complete old tuple data | Supported | Unit tests, `FuzzParseLine` seed, and PostgreSQL fixture job with `REPLICA IDENTITY FULL`. |
| Live SQL logical slot polling with `test_decoding` | Supported | `TestLiveLogicalDecoding` runs against a real `postgres:16` container in CI. |
| Wire-level logical replication with `test_decoding` | Supported | `TestWireLogicalReplication` runs against a real `postgres:16` container in CI. |
| Empty table or operation names | Rejected | Parser tests and fuzz smoke target. |
| `pgoutput` binary relation/tuple messages | Unsupported and rejected by the text parser | `TestParseLineRejectsPgoutputBinaryMessages`. |

## Live SQL Slot Reader

Register the backend and open it with a PostgreSQL DSN, a context, and a source
name matching the logical slot.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

decoder, err := registry.Open(postgres.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"),
	dblog.WithSource(dblog.Source{Name: "dblog_slot"}),
)
```

The live reader polls `pg_logical_slot_get_changes` and parses the returned
`test_decoding` text with the same parser as offline records. Cancel the context
to stop polling.

Use wire-level replication by adding `replication=database` to the DSN:

```go
decoder, err := registry.Open(postgres.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable&replication=database"),
	dblog.WithSource(dblog.Source{Name: "dblog_slot"}),
)
```

The wire reader sends `START_REPLICATION` for the slot, reads CopyData messages,
and parses the embedded `test_decoding` text with the same parser.

Both live readers are intentionally text-oriented in `v0.2.0`. Use
`test_decoding` or a custom text output plugin that can be normalized through
`decoder.WithEventPlugins`; binary `pgoutput` relation and tuple messages are
not decoded by this backend.

## Flashback Scope

| Event | Flashback output |
|---|---|
| `insert` | `DELETE FROM ... WHERE ...;` |
| `update` with complete `old-key` and `new-tuple` columns | `UPDATE ... SET old_values WHERE new_values;` |
| `delete` | `INSERT INTO ... VALUES ...;` |
| `update` without complete old/new tuple data, `begin`, `commit` | No flashback output. |

Update flashback needs enough old values to restore every new tuple column. If
the old tuple only contains key columns, the backend leaves the event out.

## Event Plugins

Use `decoder.WithEventPlugins` to handle line families outside the built-in
logical decoding text records. Plugins receive the original line after the
built-in parser declines it; keep plugin output in the backend-native
`types.Event` shape so the root adapter can preserve source, position, and
checkpoint behavior.

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
