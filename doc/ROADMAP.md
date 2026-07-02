# Roadmap

This roadmap tracks product scope for `go-dblog`. It is a release-quality
checklist, not a date commitment.

[中文](./ROADMAP.zh-CN.md)

## Status Legend

| Status | Meaning |
|---|---|
| Released | Published through GitHub Releases and git tags. |
| Done | Implemented, documented, and covered by CI. |
| Ready | Implemented and covered by CI; ready for public tags. |
| Planned | Accepted scope, not started or not complete. |
| Candidate | Useful direction, still needs design or user validation. |
| Unsupported | Explicitly not emitted or accepted in this version line. |

## Release Targets

| Version | Status | Goal | Exit gate |
|---|---|---|---|
| `v0.1.0` | Ready, superseded | First usable parser and CDC developer preview for MySQL, PostgreSQL, MongoDB, and Redis. | Implemented and CI-covered, but superseded before public tags. |
| `v0.2.0` | Released | Compatibility-hardened parser and CDC developer preview. | Protected `ci` and `merge-policy` checks passed; release tags published as `v0.2.0`, `mysql/v0.2.0`, `postgres/v0.2.0`, `mongo/v0.2.0`, and `redis/v0.2.0`. |
| `v0.3.0` | Released | Recovery workflows. | Protected `ci` and `merge-policy` checks passed; release tags published as `v0.3.0`, `mysql/v0.3.0`, `postgres/v0.3.0`, `mongo/v0.3.0`, and `redis/v0.3.0`. |
| `v0.4.0` | Ready | Operational maturity. | CI publishes a tested backend/version matrix, parser benchmark history, and structured report artifacts. |
| `v1.0.0` | Candidate | Stable public API. | Root API and backend package contracts are frozen with a documented compatibility policy. |

## Current Capability Matrix

| Capability | Common API | MySQL | PostgreSQL | MongoDB | Redis |
|---|---|---|---|---|---|
| Offline parser | N/A | Done | Done | Done | Done |
| Live reader | N/A | Done | Done | Done | Done |
| Native typed events | N/A | Done | Done | Done | Done |
| Common `dblog.Event` adapter | Done | Done | Done | Done | Done |
| Plugin hooks | N/A | Done | Done | Done | Done |
| Basic filtering | Done | Done | Done | Done | Done |
| Checkpoint/resume | Done | Done | Done | Done | Done |
| Safe flashback where the log has enough data | Done | Done | Done | Done | Done |
| Recovery plan with checkpoint handoff | Done | Done | Done | Done | Done |
| Fixture provenance | N/A | Done | Done | Done | Done |
| Malformed and unsupported input tests | Done | Done | Done | Done | Done |
| Fuzz smoke gate | N/A | Done | Done | Done | Done |
| Benchmark smoke gate | N/A | Done | Done | Done | Done |
| Static gates: lint, vet, vulnerability scan | Done | Done | Done | Done | Done |
| CI evidence artifact with tested matrix and benchmark history | Done | Done | Done | Done | Done |

## Common API

| Capability | Status | Notes |
|---|---|---|
| `dblog.Event`, `dblog.Decoder`, `dblog.Registry` | Done | Shared contracts for backend-neutral pipelines. |
| `WithReader`, `WithPath`, `WithDSN`, `WithSource`, `WithContext`, `WithCheckpoint` | Done | Common open options used by backend registry adapters. |
| Source, position, checkpoint, filtering, and flashback helpers | Done | Shared helpers keep orchestration backend-neutral. |
| Cross-database semantic normalization beyond the common event shape | Unsupported | Backend-native event bodies intentionally preserve product semantics. |
| Managed service connectors | Unsupported | Not part of the `v0.x` contract. |
| Automatic backend registration through blank imports | Unsupported | Backends register explicitly. |
| `RecoveryPlan` API and replay cookbook | Done | Streams backend-native reverse operations with source checkpoints; documented in [RECOVERY.md](./RECOVERY.md). |

CI evidence: `root_test` runs the root package tests, backend registration
tests, and checkpoint tests in every backend module.

## Operational Maturity

Detailed CI docs: [CI evidence](./CI.md).

`v0.4.0` operational work:

| Item | Status | Notes |
|---|---|---|
| Publish tested backend/version matrix in CI | Done | The `ci-report` job reads the workflow job list and publishes `tested-matrix.md` and `tested-matrix.json`. |
| Keep parser benchmark history | Done | Each parser benchmark job uploads raw benchmark output; `ci-report` consolidates it into `benchmarks.md` and `benchmarks.jsonl`. |
| Surface CI evidence in the workflow summary | Done | `ci-report` appends a Markdown summary through the GitHub Actions step summary file. |
| Treat CI evidence generation as a protected gate | Done | The aggregate `ci` job requires `ci-report` together with lint, vet, vuln, tests, fuzz, and benchmark jobs. |

## MySQL Family

Detailed user docs: [features](../mysql/doc/FEATURES.md) and
[examples](../mysql/doc/EXAMPLES.md).

| Capability | Status | Notes |
|---|---|---|
| Local MySQL-family binlog files from MySQL 5.6, 5.7, 8.0, and 8.4 | Done | CI generates real fixtures from all four images. |
| Online MySQL replication streams through `dblog.WithDSN` | Done | `TestLiveReplicationStream` runs against `mysql:8.4`. |
| MySQL, MariaDB, and MySQL-compatible binlog event bodies | Done | Event support is listed in `mysql/doc/FEATURES.md`. |
| Row event decoding through `TABLE_MAP_EVENT` metadata | Done | Missing table-map windows keep header/bitmap fields and expose `DecodeError`. |
| Built-in MariaDB plugin plus custom event plugins | Done | Plugin hooks are in `mysql/decode/decoder`. |
| Checkpoint resume through the root registry | Done | Covered by backend registry tests. |
| Safe flashback for complete write, delete, and update row images | Done | Incomplete row images are omitted. |
| GTID auto-positioning for live readers | Unsupported | Planned only after live reader compatibility policy is stable. |
| TLS-specific DSN handling | Unsupported | Not part of the `v0.3.x` contract. |
| Flashback for skipped columns or `PARTIAL_UPDATE_ROWS_EVENT` | Unsupported | Source log does not contain a complete reversible row image. |

`v0.3.0` recovery work:

| Item | Status | Notes |
|---|---|---|
| Keep existing complete row-image flashbacks | Done | Baseline from `v0.2.0`. |
| Add end-to-end recovery examples for fixture binlogs | Done | `RecoveryPlan` example shows reverse event iteration and checkpoint handoff. |
| Keep lossy row formats omitted | Done | Incomplete row images, skipped columns, and partial updates remain unsupported. |

## PostgreSQL Family

Detailed user docs: [features](../postgres/doc/FEATURES.md) and
[examples](../postgres/doc/EXAMPLES.md).

| Capability | Status | Notes |
|---|---|---|
| Logical decoding text records: `BEGIN`, `COMMIT`, and row changes | Done | Covers `test_decoding` text output. |
| Inserts, updates, and deletes in `test_decoding` text format | Done | Parser handles scalar values and quoted strings. |
| Live SQL logical slot polling through `pg_logical_slot_get_changes` | Done | `TestLiveLogicalDecoding` runs against `postgres:16`. |
| Wire-level logical replication reader for `test_decoding` | Done | `TestWireLogicalReplication` runs against `postgres:16`. |
| Event plugins for extra text line families | Done | Plugins normalize text lines after built-in parser decline. |
| Checkpoint resume through the root registry | Done | Covered by backend registry tests. |
| Safe SQL flashbacks for inserts, deletes, and complete updates | Done | Update flashback requires complete old and new tuple data. |
| `pgoutput` binary relation and tuple messages | Unsupported | Explicitly rejected by the text parser. |
| Raw WAL/page decoding | Unsupported | Outside the text logical-decoding contract. |
| Update flashback with partial old tuple data | Unsupported | Source log does not contain enough values to restore every column. |

`v0.3.0` recovery work:

| Item | Status | Notes |
|---|---|---|
| Keep SQL flashback for complete tuple records | Done | Baseline from `v0.2.0`. |
| Add recovery examples that emit reverse SQL with checkpoint state | Done | `Example_recoveryPlan` and docs cover checkpoint handoff and `REPLICA IDENTITY FULL` expectations. |
| Keep partial old-key updates omitted | Done | Partial old-key updates remain unsupported and covered by tests. |

## MongoDB Family

Detailed user docs: [features](../mongo/doc/FEATURES.md) and
[examples](../mongo/doc/EXAMPLES.md).

| Capability | Status | Notes |
|---|---|---|
| Newline-delimited oplog JSON records with `op`, `ns`, `o`, and `o2` | Done | Fixture job is generated from `mongo:7.0`. |
| Change stream JSON records with document keys, full documents, before-images, and update descriptions | Done | Malformed JSON and invalid update descriptions are rejected. |
| Live collection change streams from a MongoDB replica set | Done | `TestLiveChangeStream` runs against `mongo:7.0`. |
| Event plugins for MongoDB-compatible event shapes | Done | Plugins can normalize operations and metadata. |
| Checkpoint resume through the root registry | Done | Covered by backend registry tests. |
| Safe flashback for inserts, deletes, updates, and replacements with enough document data | Done | Updates and replacements require `fullDocumentBeforeChange`; deletes require full deleted document data. |
| Raw oplog tailing outside JSON records or change streams | Unsupported | Outside the `v0.3.x` input contract. |
| Automatic replica set or sharded cluster discovery | Unsupported | Caller supplies the DSN and source. |
| Update or replace flashback without `fullDocumentBeforeChange` | Unsupported | Source log does not contain the prior document state. |
| Delete flashback without full deleted document data | Unsupported | Source log does not contain the document to reinsert. |

`v0.3.0` recovery work:

| Item | Status | Notes |
|---|---|---|
| Keep insert/delete/update flashbacks with enough document data | Done | Baseline from `v0.2.0`. |
| Add native replace change-stream recovery when a before-image is present | Done | Covered by unit tests; no plugin is required. |
| Add live pre-image recovery examples | Done | Examples document collection pre-image requirements and `RecoveryPlan` checkpoint handoff. |

## Redis Family

Detailed user docs: [features](../redis/doc/FEATURES.md) and
[examples](../redis/doc/EXAMPLES.md).

| Capability | Status | Notes |
|---|---|---|
| Redis AOF RESP array commands | Done | Fixture job is generated from `redis:7.2`. |
| Live Redis PSYNC replication streams through `dblog.WithDSN` | Done | Live reader skips the initial RDB snapshot payload. |
| Lowercase normalized command names and native typed command events | Done | Command parser keeps original arguments. |
| Command plugins for Redis-compatible products and module commands | Done | Plugins normalize parsed commands before emission. |
| Checkpoint resume through the root registry | Done | Covered by backend registry tests. |
| Safe flashback for `LPUSH`, `RPUSH`, `INCR`, `DECR`, `INCRBY`, `DECRBY`, `HINCRBY`, `HINCRBYFLOAT`, and `ZINCRBY` | Done | Reverse commands do not require reading Redis state. |
| Redis Cluster or Sentinel discovery | Unsupported | Caller supplies a direct endpoint. |
| TLS-specific DSN handling | Unsupported | Not part of the `v0.3.x` contract. |
| Offline RDB snapshot parsing | Unsupported | Offline parser accepts RESP array command frames only. |
| Flashback for `SET`, `HSET`, `SADD`, `DEL`, and other state-dependent commands | Unsupported | Previous values, TTLs, or membership state are not available in the command log. |

`v0.3.0` recovery work:

| Item | Status | Notes |
|---|---|---|
| Keep deterministic list and counter flashbacks | Done | Baseline from `v0.2.0`. |
| Add deterministic numeric flashbacks such as `HINCRBY`, `HINCRBYFLOAT`, and `ZINCRBY` | Done | Safe because the reverse command uses the negated delta; Redis 7.2 fixture/live CI covers `HINCRBY` and `ZINCRBY`, while `HINCRBYFLOAT` is unit-tested because Redis propagates it as `HSET`. |
| Keep state-dependent commands omitted | Done | State-dependent Redis commands remain unsupported and covered by tests. |
