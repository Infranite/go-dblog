# Redis-family 功能与范围

[English](./FEATURES.md)

本文档是 Redis 族 backend 的详细功能和支持范围说明。

## 包结构

| Package | 用途 |
|---|---|
| `github.com/Infranite/go-dblog/redis` | 常用 import 的 compatibility facade。 |
| `github.com/Infranite/go-dblog/redis/backend` | 显式注册到 `dblog.Registry`。 |
| `github.com/Infranite/go-dblog/redis/decode/decoder` | 原生 streaming decoder、RESP parser 和 plugin options。 |
| `github.com/Infranite/go-dblog/redis/decode/events/types` | 原生命令、event 和 plugin types。 |

## 已支持

- Redis AOF records 的 RESP array command parsing。
- 通过 `dblog.WithDSN` 打开 live replication stream。
- 小写归一化 command name。
- Streaming RESP decoder。
- 通过 `redis/backend` 集成根 registry。
- 通过根 registry 打开时支持 `dblog.WithCheckpoint`。
- 对 list push 和确定性 numeric increment 生成无需读取 Redis state 即可安全反转的闪回命令。
- `dblog.RecoveryPlan` 会把 flashback command 和 source checkpoint 组成恢复 step。
- 面向 Redis-compatible 产品和 module commands 的 command plugin。

## 暂不支持

- Redis Cluster 或 Sentinel discovery。
- TLS-specific DSN 处理。
- 离线 RDB snapshot parsing。
- `SET`、`HSET`、`SADD`、`DEL` 等依赖旧值、TTL 或成员状态的命令闪回。

## 支持输入

| 输入 | 状态 | CI 证据 |
|---|---|---|
| Redis AOF RESP array commands | 支持 | `redis` fixture job 从 `redis:7.2` 生成；`FuzzParseCommand` smoke target。 |
| Redis replication streams | 支持 | `redis` CI job 启动 `redis:7.2`，写入 SET/INCR/LPUSH/HINCRBY/HINCRBYFLOAT/ZINCRBY，并通过 `dblog.WithDSN` 加 `dblog.WithContext` 读取传播后的 command stream。 |
| 确定性 command 的 recovery plan step | 支持 | `Example_recoveryPlan` 和 Redis fixture CI。 |
| LF-only line endings、empty command names、invalid lengths、oversized arrays/bulk strings | 拒绝 | Parser tests 和 fuzz smoke target。 |
| 离线输入中的 RDB preamble 或 mixed RDB/AOF streams | 拒绝 | `TestParseCommandRejectsInvalidRESP`。 |
| live PSYNC stream 初始 RDB snapshot payload | 读取 command 前跳过 | `TestLiveDecoderSkipsSizedRDB` 和 live Redis CI。 |
| 最多 8,192 个 RESP array elements 和 8 MiB per bulk string 的 commands | 支持 | Parser limits 由 fuzz smoke 覆盖。 |

## RDB 与混合流

离线 parser 只接收 RESP array command frames。遇到 RDB preamble 或 mixed RDB/AOF
streams 时会拒绝输入，而不是猜测 frame boundary。

live PSYNC stream 不同：Redis 会先发送一个 RDB snapshot，再发送 command stream。
live reader 会消费 snapshot payload，然后从后续 RESP command frames 开始输出事件。

## 闪回范围

| Command | 闪回输出 |
|---|---|
| `LPUSH key value ...` | `LPOP key count` |
| `RPUSH key value ...` | `RPOP key count` |
| `INCR`、`DECR`、`INCRBY`、`DECRBY` | 相反的 increment command |
| `HINCRBY key field delta`、`HINCRBYFLOAT key field delta` | 相同 command，delta 取反 |
| `ZINCRBY key delta member` | `ZINCRBY key -delta member` |

Redis 7.2 在 AOF 和 PSYNC stream 中会把 `HINCRBYFLOAT` 传播为 `HSET`。
命令级闪回实现仍支持已解析的 `HINCRBYFLOAT`，用于 Redis-compatible 产品或通过
plugin 归一化后保留原始操作的 stream。

需要 Redis 先前 state、TTL、overwritten value 或成员是否已存在的信息时，不输出闪回。
例如 `SET`、`HSET`、`SADD`、`DEL` 会被解码为 command，但不会生成闪回命令。
`dblog.RecoveryPlan` 输出相同 command，并附带原始事件 checkpoint。

## 插件支持

使用 `decoder.WithCommandPlugins` 在事件输出前归一化 Redis module commands 或
Redis-compatible dialects。插件会收到已解析的 command，并可在调用方观察到事件前改写
backend-native `types.Command` 形态。
