# Roadmap

本文档记录 `go-dblog` 的产品范围。它是 release quality checklist，不承诺具体日期。

[English](./ROADMAP.md)

## 状态说明

| 状态 | 含义 |
|---|---|
| Released | 已通过 GitHub Releases 和 git tags 发布。 |
| Done | 已实现、已文档化，并由 CI 覆盖。 |
| Ready | 已实现且由 CI 覆盖，可发布公开 tag。 |
| Planned | 已接受为规划范围，但尚未开始或尚未完成。 |
| Candidate | 有价值的方向，还需要设计或用户验证。 |
| Unsupported | 当前版本线明确不接收或不输出。 |

## 版本目标

| 版本 | 状态 | 目标 | 退出门禁 |
|---|---|---|---|
| `v0.1.0` | Ready, 已被取代 | MySQL、PostgreSQL、MongoDB、Redis 的首个可用 parser 和 CDC developer preview。 | 已实现并由 CI 覆盖，但在公开 tag 发布前被取代。 |
| `v0.2.0` | Released | 兼容性加固后的 parser 和 CDC developer preview。 | 受保护 `ci` 与 `merge-policy` 检查通过；已发布 `v0.2.0`、`mysql/v0.2.0`、`postgres/v0.2.0`、`mongo/v0.2.0`、`redis/v0.2.0` tag。 |
| `v0.3.0` | Released | 恢复工作流。 | 受保护 `ci` 与 `merge-policy` 检查通过；已发布 `v0.3.0`、`mysql/v0.3.0`、`postgres/v0.3.0`、`mongo/v0.3.0`、`redis/v0.3.0` tag。 |
| `v0.4.0` | Released | 运维成熟度。 | 受保护 `ci` 与 `merge-policy` 检查通过；CI evidence artifacts 已发布；已发布 `v0.4.0`、`mysql/v0.4.0`、`postgres/v0.4.0`、`mongo/v0.4.0`、`redis/v0.4.0` tag。 |
| `v1.0.0` | Ready | 稳定公共 API。 | API 兼容性策略见下文；公开 tag 由受保护 `ci` 和 `merge-policy` 检查门禁。 |

## v1 兼容性策略

- 根模块公开 API 在 `v1.x` 遵循 SemVer；破坏性变更需要使用 `v2` module path。
- backend module 的 public packages 在 `v1.x` 保持导出名称、option 契约和 event
  body 字段含义稳定。
- minor release 可以增加 API、新 event 字段和新数据库版本支持。
- 标记为 `Unsupported` 的能力不在兼容性承诺内，直到状态改为 `Done` 或 `Ready`。

## 当前能力矩阵

| 能力 | 公共 API | MySQL | PostgreSQL | MongoDB | Redis |
|---|---|---|---|---|---|
| 离线解析器 | N/A | Done | Done | Done | Done |
| Live reader | N/A | Done | Done | Done | Done |
| 原生 typed events | N/A | Done | Done | Done | Done |
| 公共 `dblog.Event` adapter | Done | Done | Done | Done | Done |
| 插件入口 | N/A | Done | Done | Done | Done |
| package-level logging hooks | Done | Done | Done | Done | Done |
| 基础过滤 | Done | Done | Done | Done | Done |
| Checkpoint/resume | Done | Done | Done | Done | Done |
| source log 包含足够数据时的安全闪回 | Done | Done | Done | Done | Done |
| 带 checkpoint handoff 的恢复计划 | Done | Done | Done | Done | Done |
| Fixture provenance | N/A | Done | Done | Done | Done |
| Malformed 和 unsupported input tests | Done | Done | Done | Done | Done |
| Fuzz smoke gate | N/A | Done | Done | Done | Done |
| Benchmark smoke gate | N/A | Done | Done | Done | Done |
| 静态门禁：lint、vet、vulnerability scan | Done | Done | Done | Done | Done |
| 包含已测试矩阵和 benchmark 历史的 CI evidence artifact | Done | Done | Done | Done | Done |

## 公共 API

| 能力 | 状态 | 说明 |
|---|---|---|
| `dblog.Event`、`dblog.Decoder`、`dblog.Registry` | Done | backend-neutral pipeline 的共享契约。 |
| `WithReader`、`WithPath`、`WithDSN`、`WithSource`、`WithContext`、`WithCheckpoint` | Done | backend registry adapter 共用的 open options。 |
| Source、position、checkpoint、filtering 和 flashback helpers | Done | 保持编排层 backend-neutral。 |
| `Logger`、`StdLogger` 和 package-level `Log` slots | Done | 默认使用标准库 logger，支持每包替换、调等级和 `Enabled` 判断。 |
| 超出公共事件形态的跨数据库语义归一 | Unsupported | backend-native event body 会保留产品语义。 |
| 托管服务 connector | Unsupported | 不属于 `v1.0.0` 契约。 |
| 通过 blank import 自动注册 backend | Unsupported | backend 需要显式注册。 |
| `RecoveryPlan` API 和 replay cookbook | Done | 流式输出 backend-native reverse operation 与 source checkpoint；见 [RECOVERY.zh-CN.md](./RECOVERY.zh-CN.md)。 |

CI 证据：`root_test` 运行根 package 测试；每个 backend module 都运行 backend 注册和
checkpoint 测试。

## 运维成熟度

详细 CI 文档：[CI evidence](./CI.zh-CN.md)。

`v0.4.0` 运维成熟度工作：

| 事项 | 状态 | 说明 |
|---|---|---|
| 在 CI 中发布已测试 backend/version 矩阵 | Done | `ci-report` job 读取 workflow job 列表，并发布 `tested-matrix.md` 和 `tested-matrix.json`。 |
| 保留 parser benchmark 历史 | Done | 每个 parser benchmark job 上传原始 benchmark 输出；`ci-report` 汇总为 `benchmarks.md` 和 `benchmarks.jsonl`。 |
| 在 workflow summary 展示 CI 证据 | Done | `ci-report` 通过 GitHub Actions step summary 文件追加 Markdown 摘要。 |
| 将 CI evidence 生成纳入受保护门禁 | Done | 聚合 `ci` job 同时要求 `ci-report`、lint、vet、vuln、tests、fuzz 和 benchmark jobs 通过。 |

## MySQL 族

详细用户文档：[功能](../mysql/doc/FEATURES.zh-CN.md) 和
[示例](../mysql/doc/EXAMPLES.zh-CN.md)。

| 能力 | 状态 | 说明 |
|---|---|---|
| 来自 MySQL 5.6、5.7、8.0、8.4 的本地 MySQL-family binlog 文件 | Done | CI 从四个 image 生成真实 fixture。 |
| 通过 `dblog.WithDSN` 打开在线 MySQL replication stream | Done | `TestLiveReplicationStream` 运行在 `mysql:8.4`。 |
| MySQL、MariaDB 和 MySQL-compatible binlog event bodies | Done | 事件支持列表见 `mysql/doc/FEATURES.zh-CN.md`。 |
| 基于 `TABLE_MAP_EVENT` metadata 解码 row events | Done | 缺少 table-map 窗口时保留 header/bitmap fields，并暴露 `DecodeError`。 |
| 内置 MariaDB plugin 和自定义 event plugins | Done | Plugin hooks 位于 `mysql/decode/decoder`。 |
| 通过根 registry 打开时支持 checkpoint resume | Done | backend registry tests 覆盖。 |
| 对完整 write、delete、update row image 生成安全闪回 | Done | 不完整 row image 会被省略。 |
| live reader 的 GTID auto-positioning | Unsupported | 等 live reader 兼容性策略稳定后再规划。 |
| TLS-specific DSN 处理 | Unsupported | 不属于 `v1.0.0` 契约。 |
| skipped columns 或 `PARTIAL_UPDATE_ROWS_EVENT` 的闪回 | Unsupported | source log 不包含完整可逆 row image。 |

`v0.3.0` 恢复工作：

| 事项 | 状态 | 说明 |
|---|---|---|
| 保留现有完整 row-image 闪回 | Done | `v0.2.0` 已具备的基线能力。 |
| 增加 fixture binlog 端到端恢复示例 | Done | `RecoveryPlan` 示例展示 reverse event iteration 和 checkpoint handoff。 |
| lossy row format 保持省略 | Done | incomplete row image、skipped columns 和 partial updates 仍不支持。 |

## PostgreSQL 族

详细用户文档：[功能](../postgres/doc/FEATURES.zh-CN.md) 和
[示例](../postgres/doc/EXAMPLES.zh-CN.md)。

| 能力 | 状态 | 说明 |
|---|---|---|
| logical decoding 文本记录：`BEGIN`、`COMMIT` 和 row changes | Done | 覆盖 `test_decoding` 文本输出。 |
| `test_decoding` 文本格式的 insert、update、delete | Done | parser 处理 scalar values 和 quoted strings。 |
| 通过 `pg_logical_slot_get_changes` 进行 live SQL logical slot polling | Done | `TestLiveLogicalDecoding` 运行在 `postgres:16`。 |
| 面向 `test_decoding` 的 wire-level logical replication reader | Done | `TestWireLogicalReplication` 运行在 `postgres:16`。 |
| 面向额外文本行族的 event plugins | Done | 内置 parser 不接受时由 plugin 归一化。 |
| 通过根 registry 打开时支持 checkpoint resume | Done | backend registry tests 覆盖。 |
| 对 insert、delete 和完整 update 生成安全 SQL 闪回 | Done | update 闪回要求完整 old 和 new tuple 数据。 |
| `pgoutput` binary relation 和 tuple messages | Unsupported | text parser 会明确拒绝。 |
| raw WAL/page 解码 | Unsupported | 超出文本 logical decoding 契约。 |
| partial old tuple data 的 update 闪回 | Unsupported | source log 不包含恢复每个 column 所需的值。 |

`v0.3.0` 恢复工作：

| 事项 | 状态 | 说明 |
|---|---|---|
| 保留完整 tuple records 的 SQL 闪回 | Done | `v0.2.0` 已具备的基线能力。 |
| 增加带 checkpoint state 的反向 SQL 输出恢复示例 | Done | `Example_recoveryPlan` 和文档覆盖 checkpoint handoff 与 `REPLICA IDENTITY FULL` 预期。 |
| partial old-key update 保持省略 | Done | partial old-key updates 仍不支持，并有测试覆盖。 |

## MongoDB 族

详细用户文档：[功能](../mongo/doc/FEATURES.zh-CN.md) 和
[示例](../mongo/doc/EXAMPLES.zh-CN.md)。

| 能力 | 状态 | 说明 |
|---|---|---|
| 带 `op`、`ns`、`o`、`o2` 的 newline-delimited oplog JSON records | Done | fixture job 从 `mongo:7.0` 生成。 |
| 带 document keys、full documents、before-images 和 update descriptions 的 change stream JSON records | Done | malformed JSON 和非法 update descriptions 会被拒绝。 |
| 来自 MongoDB replica set 的 live collection change streams | Done | `TestLiveChangeStream` 运行在 `mongo:7.0`。 |
| 面向 MongoDB-compatible event shape 的 event plugins | Done | plugin 可归一化 operation 和 metadata。 |
| 通过根 registry 打开时支持 checkpoint resume | Done | backend registry tests 覆盖。 |
| 输入包含足够 document data 时，为 insert、delete、update、replace 生成安全闪回 | Done | update 和 replace 需要 `fullDocumentBeforeChange`；delete 需要完整 deleted document data。 |
| JSON records 或 change streams 之外的 raw oplog tailing | Unsupported | 超出 `v1.0.0` 输入契约。 |
| 自动 replica set 或 sharded cluster discovery | Unsupported | 调用方提供 DSN 和 source。 |
| 缺少 `fullDocumentBeforeChange` 的 update 或 replace 闪回 | Unsupported | source log 不包含 prior document state。 |
| 缺少完整 deleted document data 的 delete 闪回 | Unsupported | source log 不包含可重新 insert 的 document。 |

`v0.3.0` 恢复工作：

| 事项 | 状态 | 说明 |
|---|---|---|
| 保留具备足够 document data 的 insert/delete/update 闪回 | Done | `v0.2.0` 已具备的基线能力。 |
| 当 before-image 存在时，为 replace change-stream 增加原生恢复支持 | Done | 已由单元测试覆盖；无需 plugin。 |
| 增加 live pre-image 恢复示例 | Done | examples 说明 collection pre-image 要求和 `RecoveryPlan` checkpoint handoff。 |

## Redis 族

详细用户文档：[功能](../redis/doc/FEATURES.zh-CN.md) 和
[示例](../redis/doc/EXAMPLES.zh-CN.md)。

| 能力 | 状态 | 说明 |
|---|---|---|
| Redis AOF RESP array commands | Done | fixture job 从 `redis:7.2` 生成。 |
| 通过 `dblog.WithDSN` 打开 live Redis PSYNC replication streams | Done | live reader 会跳过初始 RDB snapshot payload。 |
| 小写归一化 command name 和原生 typed command events | Done | command parser 保留原始 arguments。 |
| 面向 Redis-compatible 产品和 module commands 的 command plugins | Done | plugin 在事件输出前归一化 command。 |
| 通过根 registry 打开时支持 checkpoint resume | Done | backend registry tests 覆盖。 |
| 对 `LPUSH`、`RPUSH`、`INCR`、`DECR`、`INCRBY`、`DECRBY`、`HINCRBY`、`HINCRBYFLOAT`、`ZINCRBY` 生成安全闪回 | Done | 反向命令无需读取 Redis state。 |
| Redis Cluster 或 Sentinel discovery | Unsupported | 调用方提供 direct endpoint。 |
| TLS-specific DSN handling | Unsupported | 不属于 `v1.0.0` 契约。 |
| 离线 RDB snapshot parsing | Unsupported | 离线 parser 只接收 RESP array command frames。 |
| `SET`、`HSET`、`SADD`、`DEL` 等 state-dependent command 闪回 | Unsupported | command log 不包含 previous values、TTL 或 membership state。 |

`v0.3.0` 恢复工作：

| 事项 | 状态 | 说明 |
|---|---|---|
| 保留确定性的 list 和 counter 闪回 | Done | `v0.2.0` 已具备的基线能力。 |
| 增加 `HINCRBY`、`HINCRBYFLOAT`、`ZINCRBY` 等确定性 numeric 闪回 | Done | 使用相反 delta 构造安全反向命令；Redis 7.2 fixture/live CI 覆盖 `HINCRBY` 和 `ZINCRBY`，`HINCRBYFLOAT` 因 Redis 会传播为 `HSET`，由单元测试覆盖。 |
| state-dependent commands 保持省略 | Done | state-dependent Redis commands 仍不支持，并有测试覆盖。 |
