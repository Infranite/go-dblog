# PostgreSQL-family 功能与范围

[English](./FEATURES.md)

本文档是 PostgreSQL 族 backend 的详细功能和支持范围说明。backend driver name 是
`pg`，module path 保持 `postgres`。

## 包结构

| Package | 用途 |
|---|---|
| `github.com/Infranite/go-dblog/postgres` | 常用 import 的 compatibility facade。 |
| `github.com/Infranite/go-dblog/postgres/backend` | 显式注册到 `dblog.Registry`。 |
| `github.com/Infranite/go-dblog/postgres/decode/decoder` | 原生 streaming decoder、line parser 和 plugin options。 |
| `github.com/Infranite/go-dblog/postgres/decode/events/types` | 原生 transaction、change、event 和 plugin types。 |

## 已支持

- `BEGIN` 和 `COMMIT` transaction records。
- PostgreSQL logical decoding text form 的 row changes。
- `null`、booleans、integers、floats、quoted strings 的 scalar parsing。
- 有界 scanner buffer 的 streaming line decoder。
- PostgreSQL `test_decoding` 输出的 live SQL logical slot reader。
- PostgreSQL `test_decoding` 输出的 wire-level logical replication reader。
- 通过 `postgres/backend` 集成根 registry。
- 通过根 registry 打开时支持 `dblog.WithCheckpoint`。
- 对 insert、delete 和具备完整 old/new tuple 数据的 update 生成 SQL 闪回。
- `dblog.RecoveryPlan` 会把 reverse SQL 和 source checkpoint 组成恢复 step。
- 面向 PostgreSQL-compatible source 额外 line types 的 event plugin。

## 暂不支持

- `pgoutput` binary relation 和 tuple messages。
- raw WAL/page 解码。
- `test_decoding` 以外的 output plugin，除非自定义 text event plugin 处理。
- old tuple 未覆盖所有 new tuple column 时的 update 闪回。

## 支持输入

| 输入 | 状态 | CI 证据 |
|---|---|---|
| logical decoding text output 中的 `BEGIN` 和 `COMMIT` records | 支持 | Unit tests 和 `postgres:16` fixture job。 |
| `table schema.table: OPERATION: col[type]:value` 形式的 row changes | 支持 | Unit tests、fixture job 和 `FuzzParseLine` smoke target。 |
| `UPDATE: old-key: ... new-tuple: ...` 且具备完整 old tuple data | 支持 | Unit tests、fuzz seed 和启用 `REPLICA IDENTITY FULL` 的 fixture job。 |
| reverse SQL 与 checkpoint handoff 的 recovery plan step | 支持 | `Example_recoveryPlan`。 |
| `test_decoding` live SQL logical slot polling | 支持 | `TestLiveLogicalDecoding` 在真实 `postgres:16` 容器中运行。 |
| `test_decoding` wire-level logical replication | 支持 | `TestWireLogicalReplication` 在真实 `postgres:16` 容器中运行。 |
| empty table 或 operation names | 拒绝 | Parser tests 和 fuzz smoke target。 |
| `pgoutput` binary relation/tuple messages | 不支持，并由 text parser 拒绝 | `TestParseLineRejectsPgoutputBinaryMessages`。 |

## Live Readers

SQL reader 会轮询 `pg_logical_slot_get_changes`，并用离线 parser 解析返回的
`test_decoding` 文本。

DSN 增加 `replication=database` 时使用 wire reader。它会对配置的 slot 发送
`START_REPLICATION`，读取 CopyData messages，并解析其中的 `test_decoding` 文本。

两种 live reader 在 `v0.2.0` 都是 text-oriented：请使用 `test_decoding`，或通过
`decoder.WithEventPlugins` 归一化自定义文本 output plugin；binary `pgoutput` relation
和 tuple messages 不会被解码。

## 闪回范围

| Event | 闪回输出 |
|---|---|
| `insert` | `DELETE FROM ... WHERE ...;` |
| `update` 且包含完整 `old-key` 和 `new-tuple` columns | `UPDATE ... SET old_values WHERE new_values;` |
| `delete` | `INSERT INTO ... VALUES ...;` |
| 缺少完整 old/new tuple data 的 `update`、`begin`、`commit` | 不输出闪回。 |

Update 闪回需要足够 old values 来恢复每一个 new tuple column。如果 old tuple 只包含 key
columns，backend 会省略该闪回事件。
`dblog.RecoveryPlan` 输出相同 reverse SQL，并附带原始事件 checkpoint。

## 插件支持

使用 `decoder.WithEventPlugins` 处理内置 logical decoding 文本记录以外的 line family。
内置 parser 不接受某行后，plugin 会收到原始 line。plugin 应输出 backend-native
`types.Event`，这样 root adapter 能保留 source、position 和 checkpoint 行为。
