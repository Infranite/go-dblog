# MySQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/mysql.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/mysql)

This module is the MySQL-family backend for `go-dblog`. It decodes MySQL binary
log files and keeps MySQL, MariaDB, and MySQL-compatible dialect details in
backend-native typed events.

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need MySQL-family binlog
parsing.

## Installation

After the first `mysql/v0.1.0` tag is published:

```bash
go get github.com/Infranite/go-dblog/mysql
```

Requirements:

- Go 1.23 or later.
- A MySQL-family binary log file for the current backend.

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

## Features

- MySQL 5.1+ binlog event parsing.
- MariaDB event support through the built-in dialect plugin.
- MySQL-compatible dialect events through the MySQL-compatible decoder set.
- Row event decoding using `TABLE_MAP_EVENT` metadata.
- Future metadata-declared events preserved as `*types.MetadataEvent`.
- Go 1.23 iterator support for streaming and type filtering.
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
| Flashback for complete `WRITE_ROWS_EVENT`, `UPDATE_ROWS_EVENT`, and `DELETE_ROWS_EVENT` row images | Supported as typed reverse row events | Decoder tests and MySQL fixture CI assert emitted operations. |
| Online replication connections | Planned | Not part of the offline parser release line. |

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
describes the missing metadata.

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
patch release in that series emits the event in all configurations.

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
