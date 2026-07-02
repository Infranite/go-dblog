# Redis-family Features And Scope

[中文](./FEATURES.zh-CN.md)

This document is the detailed feature and support reference for the
Redis-family backend.

## Packages

| Package | Purpose |
|---|---|
| `github.com/Infranite/go-dblog/redis` | Compatibility facade for common imports. |
| `github.com/Infranite/go-dblog/redis/backend` | Explicit registration with `dblog.Registry`. |
| `github.com/Infranite/go-dblog/redis/decode/decoder` | Native streaming decoder, RESP parser, and plugin options. |
| `github.com/Infranite/go-dblog/redis/decode/events/types` | Native command, event, and plugin types. |

## Supported

- RESP array command parsing for Redis AOF records.
- Live replication streams opened with `dblog.WithDSN`.
- Lowercase normalized command names.
- Streaming RESP decoder.
- Root registry integration through `redis/backend`.
- Checkpoint resume through `dblog.WithCheckpoint` when opened through the root
  registry.
- Flashback commands for list pushes and deterministic numeric increments that
  can be safely reversed without reading Redis state.
- `dblog.RecoveryPlan` steps that pair flashback commands with source
  checkpoints.
- Command plugins for Redis-compatible products and module commands.

## Unsupported

- Redis Cluster or Sentinel discovery.
- TLS-specific DSN handling.
- Offline RDB snapshot parsing.
- Flashback for state-dependent commands such as `SET`, `HSET`, `SADD`, `DEL`,
  or commands that need previous values, TTLs, or membership state.

## Supported Inputs

| Input | Status | CI evidence |
|---|---|---|
| Redis AOF RESP array commands | Supported | `redis` fixture job generated from `redis:7.2`; `FuzzParseCommand` smoke target. |
| Redis replication streams | Supported | `redis` CI job starts `redis:7.2`, opens a PSYNC stream, writes SET/INCR/LPUSH/HINCRBY/HINCRBYFLOAT/ZINCRBY, and reads the propagated command stream through `dblog.WithDSN` plus `dblog.WithContext`. |
| Recovery plan steps for deterministic commands | Supported | `Example_recoveryPlan` and Redis fixture CI. |
| RESP frames with LF-only line endings, empty command names, invalid lengths, or oversized arrays/bulk strings | Rejected | Parser tests and fuzz smoke target. |
| RDB preambles or mixed RDB/AOF streams in offline input | Rejected | `TestParseCommandRejectsInvalidRESP`. |
| Initial RDB snapshot payload in live PSYNC streams | Skipped before command decoding | `TestLiveDecoderSkipsSizedRDB` and live Redis CI. |
| Commands up to 8,192 RESP array elements and 8 MiB per bulk string | Supported | Parser limits are covered by fuzz smoke. |

## RDB And Mixed Streams

The offline parser accepts RESP array command frames only. It rejects RDB
preambles and mixed RDB/AOF streams instead of guessing frame boundaries.

Live PSYNC streams are different: Redis sends an initial RDB snapshot before the
command stream. The live reader consumes that snapshot payload and starts
emitting events from the following RESP command frames.

## Flashback Scope

| Command | Flashback output |
|---|---|
| `LPUSH key value ...` | `LPOP key count` |
| `RPUSH key value ...` | `RPOP key count` |
| `INCR`, `DECR`, `INCRBY`, `DECRBY` | Opposite increment command |
| `HINCRBY key field delta`, `HINCRBYFLOAT key field delta` | Same command with the negated delta |
| `ZINCRBY key delta member` | `ZINCRBY key -delta member` |

Redis 7.2 propagates `HINCRBYFLOAT` as `HSET` in AOF and PSYNC streams. The
command-level flashback implementation still supports parsed
`HINCRBYFLOAT` commands for Redis-compatible products or plugin-normalized
streams that preserve the original operation.

Commands that require previous Redis state, TTLs, overwritten values, or
knowledge of which set/hash members already existed do not emit flashback
output. For example, `SET`, `HSET`, `SADD`, and `DEL` are decoded as commands,
but they do not produce flashback commands.
`dblog.RecoveryPlan` emits the same command plus the checkpoint of the original
event.

## Plugin Support

Use `decoder.WithCommandPlugins` to normalize Redis module commands or
Redis-compatible dialects before events are emitted. Plugins receive the parsed
command and can rewrite the backend-native `types.Command` shape before callers
observe it.
