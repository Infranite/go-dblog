# Roadmap

本文档记录 `go-dblog` 的产品范围，不承诺具体日期。

[English](./ROADMAP.md)

## 状态

| 状态 | 含义 |
|---|---|
| Done | 已实现、已文档化，并由 CI 覆盖。 |
| Ready | 已实现且由 CI 覆盖，可发布公开 tag。 |
| Planned | 已接受为规划范围，但尚未开始或尚未完成。 |
| Candidate | 有价值的方向，还需要设计或用户验证。 |
| Unsupported | 当前版本线明确不接收或不输出。 |

## 版本目标

### `v0.1.0` - Ready

目标：MySQL、PostgreSQL、MongoDB、Redis 的首个可用 parser 和 CDC developer
preview。

退出门禁：

- PR 和 `master` 上的受保护 `ci` 与 `merge-policy` 检查通过；
- 发布 `v0.1.0`、`mysql/v0.1.0`、`postgres/v0.1.0`、`mongo/v0.1.0`、
  `redis/v0.1.0` tag。

### `v0.2.0` - Planned

目标：兼容性加固。

退出门禁：每个 backend 都记录支持版本、暂不支持输入、异常输入行为和 fixture 来源。

### `v0.3.0` - Planned

目标：恢复工作流。

退出门禁：只在 source log 带有足够旧状态时扩展闪回；不安全反向操作保持省略或显式
opt-in。

### `v0.4.0` - Planned

目标：运维成熟度。

退出门禁：CI 发布已测试 backend/version 矩阵，并保留 parser benchmark 历史。

### `v1.0.0` - Candidate

目标：稳定公共 API。

退出门禁：根 API 和 backend package 契约冻结，并有兼容性策略。

## `v0.1.0` 产品范围

### 公共 API

已支持：

- `dblog.Event`、`dblog.Decoder`、`dblog.Registry` 和显式 backend 注册。
- `WithReader`、`WithPath`、`WithDSN`、`WithSource`、`WithContext`、
  `WithCheckpoint` open options。
- 公共 source、position、checkpoint、过滤和闪回辅助函数。
- backend-neutral 编排，同时保留 backend 原生事件类型。

暂不支持：

- 超出公共事件形态的跨数据库语义归一。
- 托管服务 connector。
- 通过 blank import 自动注册 backend。

CI 证据：

- `root_test` 运行 `go test -short -race -count=1 -shuffle=on ./...`。
- 每个 backend module 都运行 backend 注册和 checkpoint 测试。

### MySQL 族

已支持：

- 来自 `mysql:5.6`、`mysql:5.7`、`mysql:8.0`、`mysql:8.4` fixture 容器的本地 MySQL-family binary log 文件。
- 通过 `dblog.WithDSN` 打开在线 MySQL replication stream。
- [mysql/README.md](../mysql/README.md) 中列出的 MySQL、MariaDB 和 MySQL-compatible binlog events。
- 原生 typed event body、基于 `TABLE_MAP_EVENT` 元数据的 row event 解码，以及兼容模式中的 metadata-declared future events。
- 内置 MariaDB plugin 和自定义 event plugin。
- 通过根 registry 打开时支持 checkpoint resume。
- 对完整 write、delete、update row image 生成安全闪回。
- parser fuzz smoke 和 fixture decoder benchmark smoke 门禁。

暂不支持：

- live reader 的 GTID auto-positioning。
- TLS-specific DSN 处理。
- 输入窗口缺少 table-map metadata 时重建已解码 row value。
- incomplete row image、skipped columns 或 partial update row event 的闪回。

CI 证据：

- `mysql` job 从 `mysql:5.6`、`mysql:5.7`、`mysql:8.0`、`mysql:8.4` 生成真实 fixture。
- `TestLiveReplicationStream` 运行在 `mysql:8.4`。
- `FuzzDecodeEventHeader` 和 `BenchmarkDecoder` 作为 CI smoke gate 运行。

### PostgreSQL 族

已支持：

- PostgreSQL logical decoding 文本记录：`BEGIN`、`COMMIT` 和
  `table schema.table: OPERATION: ...` changes。
- `test_decoding` 文本格式的 insert、update、delete row changes。
- 通过 `pg_logical_slot_get_changes` 进行 live SQL logical slot 轮询。
- 面向 `test_decoding` 的 PostgreSQL wire-level replication protocol reader。
- 原生 transaction 和 change event body。
- 面向额外文本行族的 event plugin。
- 通过根 registry 打开时支持 checkpoint resume。
- 对 insert、delete 和具备完整 old/new tuple 数据的 update 生成安全 SQL 闪回。
- parser fuzz smoke 和 line parser benchmark smoke 门禁。

暂不支持：

- `pgoutput` binary relation/tuple messages。
- raw WAL/page 解码。
- `test_decoding` 以外的 output plugin，除非自定义 text event plugin 处理。
- old tuple 未覆盖所有 new tuple column 时的 update 闪回。

CI 证据：

- `postgres` job 从 `postgres:16` 生成真实 fixture。
- `TestLiveLogicalDecoding` 和 `TestWireLogicalReplication` 运行在真实 `postgres:16` 容器上。
- `FuzzParseLine` 和 `BenchmarkParseLine` 作为 CI smoke gate 运行。

### MongoDB 族

已支持：

- 按行分隔的 MongoDB oplog JSON 记录，包含 `op`、`ns`、`o`、`o2`。
- 按行分隔的 change stream JSON 记录，包含 `operationType`、`ns`、
  `documentKey`、`fullDocument`、`fullDocumentBeforeChange`、`updateDescription`。
- 来自 MongoDB replica set 的 live collection change stream。
- 原生 typed change event。
- 面向 MongoDB-compatible event shape 的 event plugin。
- 通过根 registry 打开时支持 checkpoint resume。
- 当输入包含 document key 和 before-image 数据时，为 insert、delete、update 生成安全闪回命令。
- parser fuzz smoke 和 line parser benchmark smoke 门禁。

暂不支持：

- JSON records 或 change streams 之外的 raw oplog tailing。
- 自动 replica set 或 sharded cluster discovery。
- 缺少 `fullDocumentBeforeChange` 的 update 闪回。
- 缺少完整被删除文档数据的 delete 闪回。

CI 证据：

- `mongo` job 从 `mongo:7.0` 生成真实 fixture。
- `TestLiveChangeStream` 运行在真实 `mongo:7.0` replica set。
- `FuzzParseLine` 和 `BenchmarkParseLine` 作为 CI smoke gate 运行。

### Redis 族

已支持：

- Redis AOF RESP array commands。
- 通过 `dblog.WithDSN` 打开 live Redis PSYNC replication stream。
- 小写归一化 command name 和原生 typed command event。
- 面向 Redis-compatible 产品和 module commands 的 command plugin。
- 通过根 registry 打开时支持 checkpoint resume。
- 对 `LPUSH`、`RPUSH`、`INCR`、`DECR`、`INCRBY`、`DECRBY` 族操作生成安全闪回命令。
- RESP parser fuzz smoke 和 command parser benchmark smoke 门禁。

暂不支持：

- Redis Cluster 或 Sentinel discovery。
- TLS-specific DSN 处理。
- RDB snapshot 解析。
- `SET`、`HSET`、`SADD`、`DEL` 等依赖旧值、TTL 或成员状态的命令闪回。

CI 证据：

- `redis` job 从 `redis:7.2` 生成真实 fixture。
- `TestLiveReplicationStream` 运行在真实 `redis:7.2` 服务上。
- `FuzzParseCommand` 和 `BenchmarkParseCommand` 作为 CI smoke gate 运行。

## 能力矩阵

| 能力 | MySQL | PostgreSQL | MongoDB | Redis |
|---|---|---|---|---|
| 离线解析器 | Done | Done | Done | Done |
| Live reader | Done | Done | Done | Done |
| 原生 typed events | Done | Done | Done | Done |
| 公共 `dblog.Event` adapter | Done | Done | Done | Done |
| 插件入口 | Done | Done | Done | Done |
| 基础过滤 | Done | Done | Done | Done |
| Checkpoint/resume | Done | Done | Done | Done |
| source log 包含足够数据时的安全闪回 | Done | Done | Done | Done |
| Fixture provenance | Done | Done | Done | Done |
| Fuzz smoke gate | Done | Done | Done | Done |
| Benchmark smoke gate | Done | Done | Done | Done |
| 静态门禁：lint、vet、vulnerability scan | Done | Done | Done | Done |

## 后续工作

### `v0.2.0` 兼容性加固

- MySQL：记录 live reader DSN 限制，补 incomplete table-map window 的负向 fixture，并判断 GTID auto-positioning 是否进入该版本线。
- PostgreSQL：为 `pgoutput` binary messages 增加明确 unsupported 测试，并记录 text plugin 扩展点。
- MongoDB：记录 live change stream pre-image 要求，并补 malformed JSON/update-description 负向用例。
- Redis：扩展 malformed RESP limit 测试，并记录 RDB preamble 与 mixed AOF stream 行为。

### `v0.3.0` 恢复工作流

- 只在 source log 包含足够旧状态时扩展闪回。
- 对 lossy 或 state-dependent reverse operations 保持省略，除非新增显式 opt-in API。
- checkpoint state 保持可跨进程重启迁移。

### `v0.4.0` 运维成熟度

- 每次发布时公开已测试 database/log version matrix。
- CI 保留 parser benchmark smoke gate，并记录 release-time baseline。
- 每个 backend 都要求 `govulncheck`、race test、lint 和 vet。

## 版本规则

- 根模块 tag 使用 `vX.Y.Z`。
- backend module tag 使用 module-prefixed tag：`mysql/vX.Y.Z`、
  `postgres/vX.Y.Z`、`mongo/vX.Y.Z`、`redis/vX.Y.Z`。
- `v0.x` 阶段 backend module 跟随根模块版本。
- GitHub Releases 和 git tags 是公开发布记录。
- Git history 是详细变更记录；项目不单独维护 release notes 或 changelog 文件。
