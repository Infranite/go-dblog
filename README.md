# go-mysql-binlog
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/liipx/go-mysql-binlog/blob/master/LICENSE) 
[![Go Report Card](https://goreportcard.com/badge/github.com/liipx/go-mysql-binlog)](https://goreportcard.com/report/github.com/liipx/go-mysql-binlog)


MySQL-family binary log analyzer in Golang.

[中文说明](https://github.com/liipx/go-mysql-binlog/blob/master/doc/doc_zh.md)

## Example
```go
package main

import (
	"fmt"
	"strings"

	"github.com/liipx/go-mysql-binlog/binlog/common"
	"github.com/liipx/go-mysql-binlog/binlog/decode/decoder"
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
		fmt.Printf("Got %s: \n\t", common.EventTypeName(event.Header.EventType))
		fmt.Println(event.Header)
		fmt.Println(strings.Repeat("=", 100))
	}
}
```

Typed body filtering uses Go 1.23 iterators plus generics:

```go
for queryEvent, err := range decoder.EventBodies[*types.QueryEvent](fileDecoder.Events()) {
	if err != nil {
		panic(err)
	}
	fmt.Println(queryEvent.Schema, queryEvent.Query)
}
```
### Output:
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
Got TABLE_MAP_EVENT: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:64, LogPos:362, Flag:0x0
====================================================================================================
Got WRITE_ROWS_EVENTv2: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:197, LogPos:559, Flag:0x0
====================================================================================================
Got XID_EVENT: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:31, LogPos:590, Flag:0x0
====================================================================================================

```

## Progress
The decoder targets MySQL-family binary logs: MySQL 5.1 and later, plus compatible
dialects such as MariaDB and TiDB. By default it uses `FORMAT_DESCRIPTION_EVENT`
metadata to recognize event types from different versions. Known event types are decoded
into dedicated structs; future event types declared by the metadata are preserved as
`*types.MetadataEvent` with post-header and payload split.

Use `decoder.WithEventCompatibilityMode(decoder.EventCompatibilityStrict)` to reject
event types not built into this package, or `decoder.EventCompatibilityLoose` to keep
decoding even when the metadata is incomplete.

The MariaDB plugin is enabled by default. Custom dialect extensions can register in-process
event plugins with `decoder.WithEventPlugins(...)`. Plugins are matched after
`FORMAT_DESCRIPTION_EVENT` is decoded and then merged into the decoder-local registry, so
the hot path remains a single event-type map lookup.

Row events decode column values from the latest `TABLE_MAP_EVENT` for the table id.
If decoding starts after the required table map, the row event is still returned with
header and bitmap fields populated and `BinRowsEvent.DecodeError` describing the missing
metadata.

|EventType|Supported|
|---|---|
|UNKNOWN_EVENT|✔|
|START_EVENT_V3|✔|
|QUERY_EVENT|✔|
|STOP_EVENT|✔|
|ROTATE_EVENT|✔|
|INTVAR_EVENT|✔|
|LOAD_EVENT|✔|
|SLAVE_EVENT|✔|
|CREATE_FILE_EVENT|✔|
|APPEND_BLOCK_EVENT|✔|
|EXEC_LOAD_EVENT|✔|
|DELETE_FILE_EVENT|✔|
|NEW_LOAD_EVENT|✔|
|RAND_EVENT|✔|
|USER_VAR_EVENT|✔|
|FORMAT_DESCRIPTION_EVENT|✔|
|XID_EVENT|✔|
|BEGIN_LOAD_QUERY_EVENT|✔|
|EXECUTE_LOAD_QUERY_EVENT|✔|
|TABLE_MAP_EVENT|✔|
|WRITE_ROWS_EVENTv0|✔|
|UPDATE_ROWS_EVENTv0|✔|
|DELETE_ROWS_EVENTv0|✔|
|WRITE_ROWS_EVENTv1|✔|
|UPDATE_ROWS_EVENTv1|✔|
|DELETE_ROWS_EVENTv1|✔|
|INCIDENT_EVENT|✔|
|HEARTBEAT_EVENT|✔|
|IGNORABLE_EVENT|✔|
|ROWS_QUERY_EVENT|✔|
|WRITE_ROWS_EVENTv2|✔|
|UPDATE_ROWS_EVENTv2|✔|
|DELETE_ROWS_EVENTv2|✔|
|GTID_EVENT|✔|
|ANONYMOUS_GTID_EVENT|✔|
|PREVIOUS_GTIDS_EVENT|✔|
|TRANSACTION_CONTEXT_EVENT|✔|
|VIEW_CHANGE_EVENT|✔|
|XA_PREPARE_LOG_EVENT|✔|
|PARTIAL_UPDATE_ROWS_EVENT|✔|
|TRANSACTION_PAYLOAD_EVENT|✔|
|HEARTBEAT_EVENT_V2|✔|
|GTID_TAGGED_LOG_EVENT|✔|

|MariaDB EventType|Supported|
|---|---|
|MARIADB_ANNOTATE_ROWS_EVENT|✔|
|MARIADB_BINLOG_CHECKPOINT_EVENT|✔|
|MARIADB_GTID_EVENT|✔|
|MARIADB_GTID_LIST_EVENT|✔|
|MARIADB_START_ENCRYPTION_EVENT|✔|
|MARIADB_QUERY_COMPRESSED_EVENT|✔|
|MARIADB_WRITE_ROWS_COMPRESSED_EVENT_V1|✔|
|MARIADB_UPDATE_ROWS_COMPRESSED_EVENT_V1|✔|
|MARIADB_DELETE_ROWS_COMPRESSED_EVENT_V1|✔|

TiDB replication-facing binlog events are handled by the MySQL-compatible decoder set.
There is no TiDB plugin until TiDB exposes a distinct binlog event type that needs one.

## TODO
1. Get binlog events through MySQL-family replication connections.
1. Add a concurrent binlog dumper after the network reader is stable.
1. Build flashback SQL on top of decoded row-format binary logs.
