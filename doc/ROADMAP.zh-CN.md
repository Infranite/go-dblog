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
| `v0.3.0` | Planned | 恢复工作流。 | 只在 source log 包含足够旧状态时扩展闪回；不安全反向操作保持省略或显式 opt-in；合入和打 tag 前全 backend CI 通过。 |
| `v0.4.0` | Planned | 运维成熟度。 | CI 发布已测试 backend/version 矩阵，并保留 parser benchmark 历史。 |
| `v1.0.0` | Candidate | 稳定公共 API。 | 根 API 和 backend package 契约冻结，并有兼容性策略。 |

## 当前能力矩阵

| 能力 | 公共 API | MySQL | PostgreSQL | MongoDB | Redis |
|---|---|---|---|---|---|
| 离线解析器 | N/A | Done | Done | Done | Done |
| Live reader | N/A | Done | Done | Done | Done |
| 原生 typed events | N/A | Done | Done | Done | Done |
| 公共 `dblog.Event` adapter | Done | Done | Done | Done | Done |
| 插件入口 | N/A | Done | Done | Done | Done |
| 基础过滤 | Done | Done | Done | Done | Done |
| Checkpoint/resume | Done | Done | Done | Done | Done |
| source log 包含足够数据时的安全闪回 | Done | Done | Done | Done | Done |
| Fixture provenance | N/A | Done | Done | Done | Done |
| Malformed 和 unsupported input tests | Done | Done | Done | Done | Done |
| Fuzz smoke gate | N/A | Done | Done | Done | Done |
| Benchmark smoke gate | N/A | Done | Done | Done | Done |
| 静态门禁：lint、vet、vulnerability scan | Done | Done | Done | Done | Done |

## 公共 API

| 能力 | 状态 | 说明 |
|---|---|---|
| `dblog.Event`、`dblog.Decoder`、`dblog.Registry` | Done | backend-neutral pipeline 的共享契约。 |
| `WithReader`、`WithPath`、`WithDSN`、`WithSource`、`WithContext`、`WithCheckpoint` | Done | backend registry adapter 共用的 open options。 |
| Source、position、checkpoint、filtering 和 flashback helpers | Done | 保持编排层 backend-neutral。 |
| 超出公共事件形态的跨数据库语义归一 | Unsupported | backend-native event body 会保留产品语义。 |
| 托管服务 connector | Unsupported | 不属于 `v0.x` 契约。 |
| 通过 blank import 自动注册 backend | Unsupported | backend 需要显式注册。 |
| Recovery plan API 和 replay cookbook | Planned for `v0.3.0` | 基于现有 safe flashback 和 checkpoint primitives。 |

CI 证据：`root_test` 运行根 package 测试；每个 backend module 都运行 backend 注册和
checkpoint 测试。

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
| TLS-specific DSN 处理 | Unsupported | 不属于 `v0.2.x` 契约。 |
| skipped columns 或 `PARTIAL_UPDATE_ROWS_EVENT` 的闪回 | Unsupported | source log 不包含完整可逆 row image。 |

`v0.3.0` 恢复工作：

| 事项 | 状态 | 说明 |
|---|---|---|
| 保留现有完整 row-image 闪回 | Done | `v0.2.0` 已具备的基线能力。 |
| 增加 fixture binlog 端到端恢复示例 | Planned | 需要展示 reverse event iteration 和 checkpoint handoff。 |
| lossy row format 保持省略，除非新增显式 opt-in API | Planned | 退出门禁要求。 |

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
| 增加带 checkpoint state 的反向 SQL 输出恢复示例 | Planned | 需要覆盖 `REPLICA IDENTITY FULL` 预期。 |
| partial old-key update 保持省略，除非新增显式 opt-in API | Planned | 退出门禁要求。 |

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
| JSON records 或 change streams 之外的 raw oplog tailing | Unsupported | 超出 `v0.2.x` 输入契约。 |
| 自动 replica set 或 sharded cluster discovery | Unsupported | 调用方提供 DSN 和 source。 |
| 缺少 `fullDocumentBeforeChange` 的 update 或 replace 闪回 | Unsupported | source log 不包含 prior document state。 |
| 缺少完整 deleted document data 的 delete 闪回 | Unsupported | source log 不包含可重新 insert 的 document。 |

`v0.3.0` 恢复工作：

| 事项 | 状态 | 说明 |
|---|---|---|
| 保留具备足够 document data 的 insert/delete/update 闪回 | Done | `v0.2.0` 已具备的基线能力。 |
| 当 before-image 存在时，为 replace change-stream 增加原生恢复支持 | Done | 已由单元测试覆盖；无需 plugin。 |
| 增加 live pre-image 恢复示例 | Planned | 需要说明 collection pre-image 要求。 |

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
| TLS-specific DSN handling | Unsupported | 不属于 `v0.2.x` 契约。 |
| 离线 RDB snapshot parsing | Unsupported | 离线 parser 只接收 RESP array command frames。 |
| `SET`、`HSET`、`SADD`、`DEL` 等 state-dependent command 闪回 | Unsupported | command log 不包含 previous values、TTL 或 membership state。 |

`v0.3.0` 恢复工作：

| 事项 | 状态 | 说明 |
|---|---|---|
| 保留确定性的 list 和 counter 闪回 | Done | `v0.2.0` 已具备的基线能力。 |
| 增加 `HINCRBY`、`HINCRBYFLOAT`、`ZINCRBY` 等确定性 numeric 闪回 | Done | 使用相反 delta 构造安全反向命令；fixture 和 live CI 覆盖这些命令。 |
| state-dependent commands 保持省略，除非新增显式 opt-in API | Planned | 退出门禁要求。 |
