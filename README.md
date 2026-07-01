# go-mysql-binlog

<p align="center">
  <img src="./doc/assets/title-banner.svg" alt="go-mysql-binlog title banner">
</p>

[![CI](https://github.com/Infranite/go-mysql-binlog/actions/workflows/dev-test.yml/badge.svg?branch=develop)](https://github.com/Infranite/go-mysql-binlog/actions/workflows/dev-test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Infranite/go-mysql-binlog)](https://github.com/Infranite/go-mysql-binlog/blob/develop/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-mysql-binlog.svg)](https://pkg.go.dev/github.com/Infranite/go-mysql-binlog)
[![Go Report Card](https://goreportcard.com/badge/github.com/Infranite/go-mysql-binlog)](https://goreportcard.com/report/github.com/Infranite/go-mysql-binlog)
[![License](https://img.shields.io/github/license/Infranite/go-mysql-binlog)](https://github.com/Infranite/go-mysql-binlog/blob/develop/LICENSE)

`go-mysql-binlog` is a Go library for decoding MySQL-family binary log files.
It targets MySQL 5.1 and later, MariaDB binlog extensions, and TiDB
replication-facing events that follow the MySQL binlog protocol.

[中文说明](https://github.com/Infranite/go-mysql-binlog/blob/develop/doc/doc_zh.md)

## Features

- Decode local binlog files with a streaming iterator API.
- Decode MySQL binlog events into typed event structs.
- Decode row events with `TABLE_MAP_EVENT` metadata.
- Preserve future metadata-declared events as `*types.MetadataEvent`.
- Support MariaDB-specific binlog events through the built-in plugin.
- Support custom MySQL-family dialect plugins without changing the hot path.
- Filter decoded event bodies with Go 1.23 iterators and generics.

## Requirements

- Go 1.23 or later.
- A MySQL-family binary log file.

## Installation

```bash
go get github.com/Infranite/go-mysql-binlog
```

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-mysql-binlog/binlog/common"
	"github.com/Infranite/go-mysql-binlog/binlog/decode/decoder"
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

## Typed Event Filtering

Use `decoder.EventBodies` when only one event body type matters:

```go
package main

import (
	"fmt"

	"github.com/Infranite/go-mysql-binlog/binlog/decode/decoder"
	"github.com/Infranite/go-mysql-binlog/binlog/decode/events/types"
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

## Compatibility Modes

The decoder uses `FORMAT_DESCRIPTION_EVENT` metadata to recognize event types
across MySQL versions.

```go
fileDecoder, err := decoder.NewBinFileDecoder(
	"./mysql-bin.000001",
	decoder.WithEventCompatibilityMode(decoder.EventCompatibilityLoose),
)
```

Available modes:

| Mode | Behavior |
|---|---|
| `EventCompatibilityAuto` | Accept built-in events and metadata-declared future events. |
| `EventCompatibilityStrict` | Reject event types not built into this package. |
| `EventCompatibilityLoose` | Keep decoding unknown event types as metadata events. |

## Row Events

Row events decode column values from the latest `TABLE_MAP_EVENT` for the table
id. If decoding starts after the required table map, the row event is still
returned with header and bitmap fields populated, and `BinRowsEvent.DecodeError`
describes the missing metadata.

Decoded row columns are exposed as `types.ColumnValue`. Variable-width payloads
reuse the original event buffer through `ColumnValue.Raw` to avoid unnecessary
copies.

## Dialect Plugins

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

### MySQL

| EventType | Supported |
|---|---|
| `UNKNOWN_EVENT` | Yes |
| `START_EVENT_V3` | Yes |
| `QUERY_EVENT` | Yes |
| `STOP_EVENT` | Yes |
| `ROTATE_EVENT` | Yes |
| `INTVAR_EVENT` | Yes |
| `LOAD_EVENT` | Yes |
| `SLAVE_EVENT` | Yes |
| `CREATE_FILE_EVENT` | Yes |
| `APPEND_BLOCK_EVENT` | Yes |
| `EXEC_LOAD_EVENT` | Yes |
| `DELETE_FILE_EVENT` | Yes |
| `NEW_LOAD_EVENT` | Yes |
| `RAND_EVENT` | Yes |
| `USER_VAR_EVENT` | Yes |
| `FORMAT_DESCRIPTION_EVENT` | Yes |
| `XID_EVENT` | Yes |
| `BEGIN_LOAD_QUERY_EVENT` | Yes |
| `EXECUTE_LOAD_QUERY_EVENT` | Yes |
| `TABLE_MAP_EVENT` | Yes |
| `WRITE_ROWS_EVENTv0` | Yes |
| `UPDATE_ROWS_EVENTv0` | Yes |
| `DELETE_ROWS_EVENTv0` | Yes |
| `WRITE_ROWS_EVENTv1` | Yes |
| `UPDATE_ROWS_EVENTv1` | Yes |
| `DELETE_ROWS_EVENTv1` | Yes |
| `INCIDENT_EVENT` | Yes |
| `HEARTBEAT_EVENT` | Yes |
| `IGNORABLE_EVENT` | Yes |
| `ROWS_QUERY_EVENT` | Yes |
| `WRITE_ROWS_EVENTv2` | Yes |
| `UPDATE_ROWS_EVENTv2` | Yes |
| `DELETE_ROWS_EVENTv2` | Yes |
| `GTID_EVENT` | Yes |
| `ANONYMOUS_GTID_EVENT` | Yes |
| `PREVIOUS_GTIDS_EVENT` | Yes |
| `TRANSACTION_CONTEXT_EVENT` | Yes |
| `VIEW_CHANGE_EVENT` | Yes |
| `XA_PREPARE_LOG_EVENT` | Yes |
| `PARTIAL_UPDATE_ROWS_EVENT` | Yes |
| `TRANSACTION_PAYLOAD_EVENT` | Yes |
| `HEARTBEAT_EVENT_V2` | Yes |
| `GTID_TAGGED_LOG_EVENT` | Yes |

### MariaDB

| EventType | Supported |
|---|---|
| `MARIADB_ANNOTATE_ROWS_EVENT` | Yes |
| `MARIADB_BINLOG_CHECKPOINT_EVENT` | Yes |
| `MARIADB_GTID_EVENT` | Yes |
| `MARIADB_GTID_LIST_EVENT` | Yes |
| `MARIADB_START_ENCRYPTION_EVENT` | Yes |
| `MARIADB_QUERY_COMPRESSED_EVENT` | Yes |
| `MARIADB_WRITE_ROWS_COMPRESSED_EVENT_V1` | Yes |
| `MARIADB_UPDATE_ROWS_COMPRESSED_EVENT_V1` | Yes |
| `MARIADB_DELETE_ROWS_COMPRESSED_EVENT_V1` | Yes |

## Development

Run the test suite:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Roadmap

- Read binlog events through MySQL-family replication connections.
- Add a concurrent binlog dumper after the network reader is stable.
- Build flashback SQL on top of decoded row-format binary logs.

## Contributing

Issues and pull requests are welcome. Keep changes focused, add tests for parser
behavior, and run `go test ./...` before submitting.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
