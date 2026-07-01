# go-mysql-binlog

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/liipx/go-mysql-binlog/blob/master/LICENSE) 
[![Go Report Card](https://goreportcard.com/badge/github.com/liipx/go-mysql-binlog)](https://goreportcard.com/report/github.com/liipx/go-mysql-binlog)

基于 Go 语言实现的 MySQL 族二进制日志文件解析 SDK（pre-binlog-server）。

[English](https://github.com/liipx/go-mysql-binlog/blob/master/README.md)

## 使用案例
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

基于 Go 1.23 iterator 和泛型，可以按 body 类型过滤：

```go
for queryEvent, err := range decoder.EventBodies[*types.QueryEvent](fileDecoder.Events()) {
	if err != nil {
		panic(err)
	}
	fmt.Println(queryEvent.Schema, queryEvent.Query)
}
```
### 输出
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

## 项目进度
目前并未把所有的binlog event实现完全，但每一个binlog event的读取已经做完。

解码器目标支持 MySQL 族 binlog：MySQL 5.1 及之后版本，以及 MariaDB、TiDB 等
兼容 MySQL 复制协议的方言。默认会根据 `FORMAT_DESCRIPTION_EVENT` 里的 metadata
识别不同版本的 event type。已内置的 event type 会解成专属结构体；未来版本新增但
metadata 已声明的 event type 会保留为 `*types.MetadataEvent`，并拆出 post-header
与 payload。

可以通过 `decoder.WithEventCompatibilityMode(decoder.EventCompatibilityStrict)` 拒绝
当前包尚未内置的 event type，也可以用 `decoder.EventCompatibilityLoose` 在 metadata
不完整时继续保留事件。

MariaDB 插件默认启用。其他方言扩展可以通过 `decoder.WithEventPlugins(...)`
注册进程内插件。插件会在 `FORMAT_DESCRIPTION_EVENT` 解码后匹配并合并到当前
decoder 自己的 registry，后续热路径仍然只是一次 event type map 查找。

Row event 会根据对应 table id 最近一次 `TABLE_MAP_EVENT` 解出字段值。如果从中间
offset 开始读取导致缺少 table map，row event 仍会返回 header 和 bitmap 字段，并在
`BinRowsEvent.DecodeError` 中说明缺失的 metadata，不会中断整个文件扫描。

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

TiDB 面向复制协议的 binlog event 走 MySQL-compatible 解码器集合。除非 TiDB
暴露需要单独处理的 binlog event type，否则不单独提供 TiDB 插件。

## TODO
1. 支持通过 MySQL 族复制连接获取 binlog event。
1. 网络读取稳定后，再做并发 binlog dumper。
1. 基于 row-format binary log 生成闪回 SQL。
