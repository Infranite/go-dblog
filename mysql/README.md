# MySQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/mysql.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/mysql)

This module is the MySQL-family backend for `go-dblog`. It decodes MySQL binary
log files, streams MySQL replication events, and keeps MySQL, MariaDB, and
MySQL-compatible dialect details in backend-native typed events.

[中文](./README.zh-CN.md)

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need MySQL-family binlog
parsing or replication stream reading.

## Installation

No public tags have been published yet. After the first `v0.2.0` tag set is
published:

```bash
go get github.com/Infranite/go-dblog/mysql@v0.2.0
```

The repository tag for this module is `mysql/v0.2.0`; callers use the semantic
version query above with `go get`.

Requirements:

- Go 1.25 or later.
- A MySQL-family binary log file, or a MySQL server with binary logging enabled
  and a user allowed to read replication streams.

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog/mysql/common"
	"github.com/Infranite/go-dblog/mysql/decode/decoder"
)

func main() {
	fileDecoder, err := decoder.NewBinFileDecoder("./testdata/mysql-bin.000004")
	if err != nil {
		panic(err)
	}
	defer fileDecoder.Close()

	for event, err := range fileDecoder.Events() {
		if err != nil {
			panic(err)
		}
		fmt.Printf("Got %s:\n\t", common.EventTypeName(event.Header.EventType))
		fmt.Println(event.Header)
		fmt.Println(strings.Repeat("=", 100))
	}
}
```

Example output:

```text
Got FORMAT_DESCRIPTION_EVENT:
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:119, LogPos:123, Flag:0x1
====================================================================================================
Got PREVIOUS_GTIDS_EVENT:
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:31, LogPos:154, Flag:0x80
====================================================================================================
Got ANONYMOUS_GTID_EVENT:
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:65, LogPos:219, Flag:0x0
====================================================================================================
Got QUERY_EVENT:
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:79, LogPos:298, Flag:0x8
====================================================================================================
```

## Packages

| Package | Purpose |
|---|---|
| `github.com/Infranite/go-dblog/mysql` | Compatibility facade for common imports. |
| `github.com/Infranite/go-dblog/mysql/backend` | Explicit registration with `dblog.Registry`. |
| `github.com/Infranite/go-dblog/mysql/decode/decoder` | Native file decoder, live replication reader, and parser options. |
| `github.com/Infranite/go-dblog/mysql/decode/events/types` | Native binlog event and plugin types. |

## Features

- MySQL 5.1+ binlog event parsing.
- MariaDB event support through the built-in dialect plugin.
- MySQL-compatible dialect events through the MySQL-compatible decoder set.
- Row event decoding using `TABLE_MAP_EVENT` metadata.
- Future metadata-declared events preserved as `*types.MetadataEvent`.
- Go iterator support for streaming and type filtering.
- Live replication streams opened with `dblog.WithDSN`.
- Copy-aware row value decoding for variable-width payloads.
- Checkpoint resume through `dblog.WithCheckpoint` when opened through the root
  registry.
- Typed flashback row events for complete write, update, and delete row images.

## Supported Inputs

| Input | Status | CI evidence |
|---|---|---|
| Local MySQL-family binlog files from MySQL 5.6, 5.7, 8.0, and 8.4 | Supported | `mysql` CI matrix generates real binlog fixtures from each image. |
| MySQL, MariaDB, and MySQL-compatible binlog event bodies listed below | Supported | Unit tests cover event decoders and the MariaDB plugin. |
| Unknown events declared by `FORMAT_DESCRIPTION_EVENT` metadata | Supported as metadata events in auto/loose compatibility modes | Compatibility mode tests and fixture tests. |
| Malformed or undersized event headers | Rejected | `FuzzDecodeEventHeader` smoke target. |
| Row events decoded without the required prior `TABLE_MAP_EVENT` | Returned with `DecodeError` and without reconstructed row values | `TestRowsEventWithoutPriorTableMapKeepsDecodeError`. |
| Flashback for complete `WRITE_ROWS_EVENT`, `UPDATE_ROWS_EVENT`, and `DELETE_ROWS_EVENT` row images | Supported as typed reverse row events | Decoder tests and MySQL fixture CI assert emitted operations. |
| Online replication connections | Supported | `TestLiveReplicationStream` runs against a real `mysql:8.4` container in CI. |

### Live Replication Reader

Register the backend and open it with a MySQL DSN and a context.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

decoder, err := registry.Open(mysql.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("mysql://dblog:dblog@127.0.0.1:3306/?server_id=1002"),
)
```

The DSN supports optional `binlog` or `file` and `pos` query parameters. When
they are omitted, the reader starts from the server's current binary log
position. GTID auto-positioning and TLS-specific DSN handling are outside the
`v0.2.0` contract. Cancel the context to stop the stream. Row details require
row-based binary logging.

### Typed Event Filtering

Use `decoder.EventBodies` when only one event body type matters:

```go
package main

import (
	"fmt"

	"github.com/Infranite/go-dblog/mysql/decode/decoder"
	"github.com/Infranite/go-dblog/mysql/decode/events/types"
)

func main() {
	fileDecoder, err := decoder.NewBinFileDecoder("./testdata/mysql-bin.000004")
	if err != nil {
		panic(err)
	}
	defer fileDecoder.Close()

	for queryEvent, err := range decoder.EventBodies[*types.QueryEvent](fileDecoder.Events()) {
		if err != nil {
			panic(err)
		}
		fmt.Println(queryEvent.Schema, queryEvent.Query)
	}
}
```

### Compatibility Modes

The MySQL backend uses `FORMAT_DESCRIPTION_EVENT` metadata to recognize event
types across MySQL versions.

```go
fileDecoder, err := decoder.NewBinFileDecoder(
	"./mysql-bin.000001",
	decoder.WithEventCompatibilityMode(decoder.EventCompatibilityLoose),
)
```

| Mode | Behavior |
|---|---|
| `EventCompatibilityAuto` | Accept built-in events and metadata-declared future events. |
| `EventCompatibilityStrict` | Reject event types not built into this package. |
| `EventCompatibilityLoose` | Keep decoding unknown event types as metadata events. |

### Row Events

Row events decode column values from the latest `TABLE_MAP_EVENT` for the table
id. If decoding starts after the required table map, the row event is still
returned with header and bitmap fields populated, and `BinRowsEvent.DecodeError`
describes the missing metadata. The decoder does not guess column values from an
incomplete input window.

Decoded row columns are exposed as `types.ColumnValue`. The rows event also
carries the schema and table name from the matching table map. Variable-width
payloads reuse the original event buffer through `ColumnValue.Raw` to avoid
unnecessary copies.

### Flashback Scope

`dblog.Flashbacks` emits synthetic `*events.Event` values with typed
`*types.BinRowsEvent` bodies when the original rows event carries a complete row
image.

| Original event | Flashback event |
|---|---|
| `WRITE_ROWS_EVENTv0/v1/v2` | Matching-version `DELETE_ROWS_EVENT` |
| `DELETE_ROWS_EVENTv0/v1/v2` | Matching-version `WRITE_ROWS_EVENT` |
| `UPDATE_ROWS_EVENTv0/v1/v2` | Matching-version `UPDATE_ROWS_EVENT` with before/after rows swapped |

Rows events with missing table-map metadata, skipped columns, or
`PARTIAL_UPDATE_ROWS_EVENT` do not emit flashback output.

### Dialect Plugins

The MariaDB plugin is enabled by default. It registers MariaDB event types after
the decoder sees a MariaDB `FORMAT_DESCRIPTION_EVENT`.

Custom MySQL-family dialects can register event plugins:

```go
fileDecoder, err := decoder.NewBinFileDecoder(
	"./mysql-bin.000001",
	decoder.WithEventPlugins(myPlugin),
)
```

TiDB replication-facing binlog events are handled by the MySQL-compatible decoder
set. There is no TiDB plugin until TiDB exposes a distinct binlog event type that
needs one.

## Event Support

The event tables describe the currently implemented MySQL-family backend. The
"First seen" column is a practical compatibility guide, not a promise that every
patch version in that series emits the event in all configurations.

### MySQL

| EventType | First seen | Supported |
|---|---:|---|
| `UNKNOWN_EVENT` | Protocol placeholder | Yes |
| `START_EVENT_V3` | pre-5.0 | Yes |
| `QUERY_EVENT` | pre-5.0 | Yes |
| `STOP_EVENT` | pre-5.0 | Yes |
| `ROTATE_EVENT` | pre-5.0 | Yes |
| `INTVAR_EVENT` | pre-5.0 | Yes |
| `LOAD_EVENT` | pre-5.0 | Yes |
| `SLAVE_EVENT` | pre-5.0 | Yes |
| `CREATE_FILE_EVENT` | pre-5.0 | Yes |
| `APPEND_BLOCK_EVENT` | pre-5.0 | Yes |
| `EXEC_LOAD_EVENT` | pre-5.0 | Yes |
| `DELETE_FILE_EVENT` | pre-5.0 | Yes |
| `NEW_LOAD_EVENT` | pre-5.0 | Yes |
| `RAND_EVENT` | pre-5.0 | Yes |
| `USER_VAR_EVENT` | pre-5.0 | Yes |
| `FORMAT_DESCRIPTION_EVENT` | 5.0.0 | Yes |
| `XID_EVENT` | 5.0.0 | Yes |
| `BEGIN_LOAD_QUERY_EVENT` | 5.0.0 | Yes |
| `EXECUTE_LOAD_QUERY_EVENT` | 5.0.0 | Yes |
| `TABLE_MAP_EVENT` | 5.1.5 | Yes |
| `WRITE_ROWS_EVENTv0` | 5.1.5 | Yes |
| `UPDATE_ROWS_EVENTv0` | 5.1.5 | Yes |
| `DELETE_ROWS_EVENTv0` | 5.1.5 | Yes |
| `WRITE_ROWS_EVENTv1` | 5.1.16 | Yes |
| `UPDATE_ROWS_EVENTv1` | 5.1.16 | Yes |
| `DELETE_ROWS_EVENTv1` | 5.1.16 | Yes |
| `INCIDENT_EVENT` | 5.1 | Yes |
| `HEARTBEAT_EVENT` | 5.1 | Yes |
| `IGNORABLE_EVENT` | 5.1 | Yes |
| `ROWS_QUERY_EVENT` | 5.6.2 | Yes |
| `WRITE_ROWS_EVENTv2` | 5.6.6 | Yes |
| `UPDATE_ROWS_EVENTv2` | 5.6.6 | Yes |
| `DELETE_ROWS_EVENTv2` | 5.6.6 | Yes |
| `GTID_EVENT` | 5.6 | Yes |
| `ANONYMOUS_GTID_EVENT` | 5.6 | Yes |
| `PREVIOUS_GTIDS_EVENT` | 5.6 | Yes |
| `TRANSACTION_CONTEXT_EVENT` | 5.7.17 | Yes |
| `VIEW_CHANGE_EVENT` | 5.7.17 | Yes |
| `XA_PREPARE_LOG_EVENT` | 5.7.7 | Yes |
| `PARTIAL_UPDATE_ROWS_EVENT` | 8.0.3 | Yes |
| `TRANSACTION_PAYLOAD_EVENT` | 8.0.20 | Yes |
| `HEARTBEAT_EVENT_V2` | 8.0.28 | Yes |
| `GTID_TAGGED_LOG_EVENT` | 8.4.0 | Yes |

### MariaDB

| EventType | First seen | Supported |
|---|---:|---|
| `MARIADB_ANNOTATE_ROWS_EVENT` | 10.0 | Yes |
| `MARIADB_BINLOG_CHECKPOINT_EVENT` | 10.0 | Yes |
| `MARIADB_GTID_EVENT` | 10.0 | Yes |
| `MARIADB_GTID_LIST_EVENT` | 10.0 | Yes |
| `MARIADB_START_ENCRYPTION_EVENT` | 10.1.7 | Yes |
| `MARIADB_QUERY_COMPRESSED_EVENT` | 10.2 | Yes |
| `MARIADB_WRITE_ROWS_COMPRESSED_EVENT_V1` | 10.2 | Yes |
| `MARIADB_UPDATE_ROWS_COMPRESSED_EVENT_V1` | 10.2 | Yes |
| `MARIADB_DELETE_ROWS_COMPRESSED_EVENT_V1` | 10.2 | Yes |

## Development

From the repository root, run the MySQL backend test suite:

```bash
cd mysql && GOWORK=off go test ./...
```

Run tests with coverage:

```bash
cd mysql && GOWORK=off go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Generate a real MySQL binlog fixture when you want the integration tests to run
locally:

```bash
make integration-mysql
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
