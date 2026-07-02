# Roadmap

This roadmap tracks product scope for `go-dblog`. It is not a date commitment.

[中文](./ROADMAP.zh-CN.md)

## Status

| Status | Meaning |
|---|---|
| Done | Implemented, documented, and covered by CI. |
| Ready | Implemented and covered by CI; ready for a public tag. |
| Planned | Accepted scope, not started or not complete. |
| Candidate | Useful direction, still needs design or user validation. |
| Unsupported | Explicitly not emitted or accepted in this version line. |

## Release Targets

### `v0.1.0` - Ready

Goal: first usable parser and CDC developer preview for MySQL, PostgreSQL,
MongoDB, and Redis.

Exit gate:

- protected `ci` and `merge-policy` checks pass on a PR and on `master`;
- release tags are published as `v0.1.0`, `mysql/v0.1.0`,
  `postgres/v0.1.0`, `mongo/v0.1.0`, and `redis/v0.1.0`.

### `v0.2.0` - Planned

Goal: compatibility hardening.

Exit gate: each backend documents supported versions, unsupported inputs,
malformed-input behavior, and fixture provenance.

### `v0.3.0` - Planned

Goal: recovery workflows.

Exit gate: flashback behavior is expanded only where the source log contains
enough prior state; unsafe reverse operations stay omitted or opt-in.

### `v0.4.0` - Planned

Goal: operational maturity.

Exit gate: CI publishes the tested backend/version matrix and keeps parser
benchmark history.

### `v1.0.0` - Candidate

Goal: stable public API.

Exit gate: root API and backend package contracts are frozen with a documented
compatibility policy.

## Product Scope For `v0.1.0`

### Common API

Supported now:

- `dblog.Event`, `dblog.Decoder`, `dblog.Registry`, and explicit backend
  registration.
- `WithReader`, `WithPath`, `WithDSN`, `WithSource`, `WithContext`, and
  `WithCheckpoint` open options.
- Shared source, position, checkpoint, filtering, and flashback helpers.
- Backend-neutral orchestration without hiding backend-native event types.

Not supported now:

- Cross-database semantic normalization beyond the common event shape.
- Managed service connectors.
- Automatic backend registration through blank imports.

CI evidence:

- `root_test` runs `go test -short -race -count=1 -shuffle=on ./...`.
- Backend registration and checkpoint tests run in every backend module.

### MySQL Family

Supported now:

- Local MySQL-family binary log files from MySQL `5.6`, `5.7`, `8.0`, and
  `8.4` fixture containers.
- Online MySQL replication streams through `dblog.WithDSN`.
- MySQL, MariaDB, and MySQL-compatible binlog events listed in
  [mysql/README.md](../mysql/README.md).
- Native typed event bodies, row event decoding through `TABLE_MAP_EVENT`
  metadata, and metadata-declared future events in compatibility modes.
- Built-in MariaDB plugin plus custom event plugins.
- Checkpoint resume when opened through the root registry.
- Safe flashback for complete write, delete, and update row images.
- Parser fuzz smoke and fixture decoder benchmark smoke gates.

Not supported now:

- GTID auto-positioning for live readers.
- TLS-specific DSN handling.
- Reconstructing decoded row values when the required table-map metadata is
  missing from the input window.
- Flashback for incomplete row images, skipped columns, or partial update row
  events.

CI evidence:

- The `mysql` job generates real fixtures from `mysql:5.6`, `mysql:5.7`,
  `mysql:8.0`, and `mysql:8.4`.
- `TestLiveReplicationStream` runs against `mysql:8.4`.
- `FuzzDecodeEventHeader` and `BenchmarkDecoder` run as CI smoke gates.

### PostgreSQL Family

Supported now:

- PostgreSQL logical decoding text records: `BEGIN`, `COMMIT`, and
  `table schema.table: OPERATION: ...` changes.
- Row changes for inserts, updates, and deletes in `test_decoding` text format.
- Live SQL logical slot polling through `pg_logical_slot_get_changes`.
- Wire-level PostgreSQL replication protocol reader for `test_decoding`.
- Native transaction and change event bodies.
- Event plugins for extra text line families.
- Checkpoint resume when opened through the root registry.
- Safe SQL flashbacks for inserts, deletes, and updates with complete old and
  new tuple data.
- Parser fuzz smoke and line parser benchmark smoke gates.

Not supported now:

- `pgoutput` binary relation/tuple messages.
- Raw WAL/page decoding.
- Output plugins outside `test_decoding` unless a custom text event plugin
  handles them.
- Update flashback when the old tuple does not cover every new tuple column.

CI evidence:

- The `postgres` job generates a real fixture from `postgres:16`.
- `TestLiveLogicalDecoding` and `TestWireLogicalReplication` run against a real
  `postgres:16` container.
- `FuzzParseLine` and `BenchmarkParseLine` run as CI smoke gates.

### MongoDB Family

Supported now:

- Newline-delimited MongoDB oplog JSON records with `op`, `ns`, `o`, and `o2`.
- Newline-delimited change stream JSON records with `operationType`, `ns`,
  `documentKey`, `fullDocument`, `fullDocumentBeforeChange`, and
  `updateDescription`.
- Live collection change streams from a MongoDB replica set.
- Native typed change events.
- Event plugins for MongoDB-compatible event shapes.
- Checkpoint resume when opened through the root registry.
- Safe flashback commands for inserts, deletes, and updates that include a
  document key and before-image data.
- Parser fuzz smoke and line parser benchmark smoke gates.

Not supported now:

- Raw oplog tailing outside JSON records or change streams.
- Automatic replica set or sharded cluster discovery.
- Update flashback without `fullDocumentBeforeChange`.
- Delete flashback without full deleted document data.

CI evidence:

- The `mongo` job generates a real fixture from `mongo:7.0`.
- `TestLiveChangeStream` runs against a real `mongo:7.0` replica set.
- `FuzzParseLine` and `BenchmarkParseLine` run as CI smoke gates.

### Redis Family

Supported now:

- Redis AOF RESP array commands.
- Live Redis PSYNC replication streams through `dblog.WithDSN`.
- Lowercase normalized command names and native typed command events.
- Command plugins for Redis-compatible products and module commands.
- Checkpoint resume when opened through the root registry.
- Safe flashback commands for `LPUSH`, `RPUSH`, `INCR`, `DECR`, `INCRBY`, and
  `DECRBY`-family operations.
- RESP parser fuzz smoke and command parser benchmark smoke gates.

Not supported now:

- Redis Cluster or Sentinel discovery.
- TLS-specific DSN handling.
- RDB snapshot parsing.
- Flashback for state-dependent commands such as `SET`, `HSET`, `SADD`, `DEL`,
  or commands that need previous values, TTLs, or membership state.

CI evidence:

- The `redis` job generates a real fixture from `redis:7.2`.
- `TestLiveReplicationStream` runs against a real `redis:7.2` server.
- `FuzzParseCommand` and `BenchmarkParseCommand` run as CI smoke gates.

## Capability Matrix

| Capability | MySQL | PostgreSQL | MongoDB | Redis |
|---|---|---|---|---|
| Offline parser | Done | Done | Done | Done |
| Live reader | Done | Done | Done | Done |
| Native typed events | Done | Done | Done | Done |
| Common `dblog.Event` adapter | Done | Done | Done | Done |
| Plugin hooks | Done | Done | Done | Done |
| Basic filtering | Done | Done | Done | Done |
| Checkpoint/resume | Done | Done | Done | Done |
| Safe flashback where the log has enough data | Done | Done | Done | Done |
| Fixture provenance | Done | Done | Done | Done |
| Fuzz smoke gate | Done | Done | Done | Done |
| Benchmark smoke gate | Done | Done | Done | Done |
| Static gates: lint, vet, vulnerability scan | Done | Done | Done | Done |

## Next Work

### `v0.2.0` Compatibility Hardening

- MySQL: document live reader DSN limits, add negative fixtures for incomplete
  table-map windows, and decide whether GTID auto-positioning belongs in this
  version line.
- PostgreSQL: add explicit unsupported tests for `pgoutput` binary messages and
  document text plugin extension points.
- MongoDB: document live change stream pre-image requirements and add malformed
  JSON/update-description negative cases.
- Redis: expand malformed RESP limit tests and document behavior around RDB
  preambles and mixed AOF streams.

### `v0.3.0` Recovery Workflows

- Expand flashback only for operations with enough prior state in the source log.
- Keep lossy or state-dependent reverse operations omitted unless an explicit
  opt-in API is added.
- Keep checkpoint state portable across process restarts.

### `v0.4.0` Operations

- Publish the tested database/log version matrix with each release.
- Keep parser benchmark smoke gates in CI and record release-time baselines.
- Keep `govulncheck`, race tests, lint, and vet required for every backend.

## Versioning Rules

- Root module tags use `vX.Y.Z`.
- Backend module tags use module-prefixed tags: `mysql/vX.Y.Z`,
  `postgres/vX.Y.Z`, `mongo/vX.Y.Z`, and `redis/vX.Y.Z`.
- Backend modules track the root module version for `v0.x` tags.
- GitHub Releases and git tags are the public release record.
- Git history is the detailed change log; this repository does not maintain
  separate release notes or changelog files.
