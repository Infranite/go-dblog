# MySQL-family 功能与范围

[English](./FEATURES.md)

本文档是 MySQL 族 backend 的详细功能和支持范围说明。

## 包结构

| Package | 用途 |
|---|---|
| `github.com/Infranite/go-dblog/mysql` | 常用 import 的 compatibility facade。 |
| `github.com/Infranite/go-dblog/mysql/backend` | 显式注册到 `dblog.Registry`。 |
| `github.com/Infranite/go-dblog/mysql/decode/decoder` | 原生 file decoder、live replication reader 和 parser options。 |
| `github.com/Infranite/go-dblog/mysql/decode/events/types` | 原生 binlog event 和 plugin types。 |

## 已支持

- MySQL 5.1+ binlog event parsing。
- 从 MySQL `5.6`、`5.7`、`8.0`、`8.4` 生成的本地 binlog fixtures。
- 通过 `dblog.WithDSN` 打开 live replication stream。
- live reader 支持可选 `binlog` 或 `file` 以及 `pos` DSN query 参数。
- 通过内置 dialect plugin 支持 MariaDB events。
- MySQL-compatible dialect events。
- 基于 `TABLE_MAP_EVENT` metadata 解码 row events。
- 在 auto/loose compatibility mode 中将 metadata-declared future events 保留为
  `*types.MetadataEvent`。
- Go iterator streaming 和 typed event filtering。
- variable-width payload 的 copy-aware row value decoding。
- 通过根 registry 打开时支持 `dblog.WithCheckpoint`。
- 对完整 write、update、delete row image 输出 typed flashback row events。
- `dblog.RecoveryPlan` 会把 reverse row event 和 source checkpoint 组成恢复 step。

## 暂不支持

- live reader 的 GTID auto-positioning。
- TLS-specific DSN 处理。
- 输入窗口缺少 table-map metadata 时重建已解码 row value。
- incomplete row image、skipped columns 或 `PARTIAL_UPDATE_ROWS_EVENT` 的闪回。

## 支持输入

| 输入 | 状态 | CI 证据 |
|---|---|---|
| 来自 MySQL 5.6、5.7、8.0、8.4 的本地 MySQL-family binlog 文件 | 支持 | `mysql` CI matrix 从每个 image 生成真实 binlog fixture。 |
| 下方列出的 MySQL、MariaDB 和 MySQL-compatible binlog event body | 支持 | Unit tests 覆盖 event decoders 和 MariaDB plugin。 |
| `FORMAT_DESCRIPTION_EVENT` metadata 声明的 unknown events | auto/loose compatibility mode 中作为 metadata events 保留 | Compatibility mode tests 和 fixture tests。 |
| malformed 或 undersized event headers | 拒绝 | `FuzzDecodeEventHeader` smoke target。 |
| 缺少前置 `TABLE_MAP_EVENT` 时解码 row events | 返回 event，并在 `DecodeError` 中记录原因，不重建 row value | `TestRowsEventWithoutPriorTableMapKeepsDecodeError`。 |
| 完整 `WRITE_ROWS_EVENT`、`UPDATE_ROWS_EVENT`、`DELETE_ROWS_EVENT` row image 的闪回 | 支持 typed reverse row events | Decoder tests 和 MySQL fixture CI。 |
| 完整 row-image 闪回的 recovery plan step | 支持 | `TestDblogRecoveryPlanIncludesCheckpoint`。 |
| 在线 replication connection | 支持 | `TestLiveReplicationStream` 在 CI 中运行真实 `mysql:8.4` 容器。 |

## Live Replication Reader

注册 backend，并用 MySQL DSN 和 context 打开。DSN 支持可选 `binlog` 或 `file` 以及
`pos` query 参数。省略时从 server 当前 binary log position 开始。取消 context 可停止
stream。Row details 需要 row-based binary logging。

GTID auto-positioning 和 TLS-specific DSN 处理不属于 `v1.0.0` 契约。

## Compatibility Modes

MySQL backend 使用 `FORMAT_DESCRIPTION_EVENT` metadata 识别不同 MySQL 版本的 event
types。

| Mode | 行为 |
|---|---|
| `EventCompatibilityAuto` | 接收内置 event 和 metadata-declared future events。 |
| `EventCompatibilityStrict` | 拒绝未内置在本包中的 event type。 |
| `EventCompatibilityLoose` | 将 unknown event type 保留为 metadata event。 |

## Row Events

Row events 从对应 table id 最新的 `TABLE_MAP_EVENT` 解码 column values。如果解码窗口
从 required table map 之后开始，row event 仍会返回 header 和 bitmap fields，
`BinRowsEvent.DecodeError` 会描述缺失 metadata。decoder 不会从不完整输入窗口猜测
column values。

已解码 column 暴露为 `types.ColumnValue`。rows event 还携带 matching table map 中的
schema 和 table name。Variable-width payload 通过 `ColumnValue.Raw` 复用原始 event
buffer，避免不必要的 copy。

## 闪回范围

`dblog.Flashbacks` 在原始 rows event 携带完整 row image 时输出 synthetic
`*events.Event`，body 为 typed `*types.BinRowsEvent`。

`dblog.RecoveryPlan` 输出相同 reverse row event，并附带原始事件 checkpoint。

| 原始 event | 闪回 event |
|---|---|
| `WRITE_ROWS_EVENTv0/v1/v2` | 同版本 `DELETE_ROWS_EVENT` |
| `DELETE_ROWS_EVENTv0/v1/v2` | 同版本 `WRITE_ROWS_EVENT` |
| `UPDATE_ROWS_EVENTv0/v1/v2` | before/after rows 交换后的同版本 `UPDATE_ROWS_EVENT` |

缺少 table-map metadata、skipped columns 或 `PARTIAL_UPDATE_ROWS_EVENT` 不输出闪回。

## 插件支持

MariaDB plugin 默认启用。decoder 看到 MariaDB `FORMAT_DESCRIPTION_EVENT` 后会注册
MariaDB event types。

自定义 MySQL-family 方言可以通过 `decoder.WithEventPlugins` 注册 event plugin。

TiDB replication-facing binlog events 由 MySQL-compatible decoder set 处理。除非 TiDB
暴露需要单独处理的 distinct binlog event type，否则不需要 TiDB plugin。

## 事件支持

事件表描述当前已经实现的 MySQL 族 backend。"First seen" 是实用兼容性参考，不承诺每个
patch version 在所有配置下都会发出该 event。

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
