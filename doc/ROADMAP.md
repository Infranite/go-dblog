# Roadmap

This roadmap tracks product scope for `go-dblog`. It is a release-quality
checklist, not a date commitment.

[中文](./ROADMAP.zh-CN.md)

## Status

| Status | Meaning |
|---|---|
| Done | Implemented, documented, and covered by CI. |
| Ready | Implemented and covered by CI; ready for public tags. |
| Planned | Accepted scope, not started or not complete. |
| Candidate | Useful direction, still needs design or user validation. |
| Unsupported | Explicitly not emitted or accepted in this version line. |

## Release Targets

### `v0.1.0` - Ready, superseded

Goal: first usable parser and CDC developer preview for MySQL, PostgreSQL,
MongoDB, and Redis.

Status: the scope is implemented and CI-covered, but the project has no public
tags yet. The first public tag set should use the `v0.2.0` target below instead
of publishing this superseded target.

### `v0.2.0` - Ready

Goal: compatibility-hardened parser and CDC developer preview.

Exit gate:

- protected `ci` and `merge-policy` checks pass on a PR and on `master`;
- each backend documents supported versions, unsupported inputs,
  malformed-input behavior, and fixture provenance;
- compatibility hardening has negative tests for incomplete metadata windows,
  unsupported binary formats, malformed JSON/update descriptions, RESP limits,
  and RDB-prefixed Redis streams;
- release tags can be published as `v0.2.0`, `mysql/v0.2.0`,
  `postgres/v0.2.0`, `mongo/v0.2.0`, and `redis/v0.2.0`.

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

## Product Scope For `v0.2.0`

### Common API

Supported:

- `dblog.Event`, `dblog.Decoder`, `dblog.Registry`, and explicit backend
  registration.
- `WithReader`, `WithPath`, `WithDSN`, `WithSource`, `WithContext`, and
  `WithCheckpoint` open options.
- Shared source, position, checkpoint, filtering, and flashback helpers.
- Backend-neutral orchestration without hiding backend-native event types.

Unsupported:

- Cross-database semantic normalization beyond the common event shape.
- Managed service connectors.
- Automatic backend registration through blank imports.

CI evidence:

- `root_test` runs `go test -short -race -count=1 -shuffle=on ./...`.
- Backend registration and checkpoint tests run in every backend module.

### MySQL Family

Supported:

- Local MySQL-family binary log files from MySQL `5.6`, `5.7`, `8.0`, and
  `8.4` fixture containers.
- Online MySQL replication streams through `dblog.WithDSN`.
- Optional live-reader `binlog` or `file` and `pos` DSN query parameters.
- MySQL, MariaDB, and MySQL-compatible binlog events listed in
  [mysql/README.md](../mysql/README.md).
- Native typed event bodies, row event decoding through `TABLE_MAP_EVENT`
  metadata, and metadata-declared future events in compatibility modes.
- Built-in MariaDB plugin plus custom event plugins.
- Checkpoint resume when opened through the root registry.
- Safe flashback for complete write, delete, and update row images.
- Parser fuzz smoke and fixture decoder benchmark smoke gates.

Compatibility behavior:

- A row event that is decoded without the required prior `TABLE_MAP_EVENT` is
  still returned with header and bitmap fields, and its `DecodeError` describes
  the missing metadata.
- Malformed or undersized event headers are rejected.

Unsupported:

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
- `TestRowsEventWithoutPriorTableMapKeepsDecodeError` covers incomplete
  table-map input windows.
- `FuzzDecodeEventHeader` and `BenchmarkDecoder` run as CI smoke gates.

### PostgreSQL Family

Supported:

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

Compatibility behavior:

- `pgoutput` binary relation and tuple messages are explicitly rejected by the
  text parser.
- Live readers parse `test_decoding` text output only; other text output
  families must be normalized by custom event plugins.

Unsupported:

- `pgoutput` binary relation/tuple messages.
- Raw WAL/page decoding.
- Output plugins outside `test_decoding` unless a custom text event plugin
  handles them.
- Update flashback when the old tuple does not cover every new tuple column.

CI evidence:

- The `postgres` job generates a real fixture from `postgres:16`.
- `TestLiveLogicalDecoding` and `TestWireLogicalReplication` run against a real
  `postgres:16` container.
- `TestParseLineRejectsPgoutputBinaryMessages` covers unsupported binary
  `pgoutput` messages.
- `FuzzParseLine` and `BenchmarkParseLine` run as CI smoke gates.

### MongoDB Family

Supported:

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

Compatibility behavior:

- Malformed JSON records are rejected.
- Change stream `updateDescription`, when present, must be a JSON object.
- Live update flashback requires `fullDocumentBeforeChange`; users must enable
  MongoDB change stream pre-images on the source collection.

Unsupported:

- Raw oplog tailing outside JSON records or change streams.
- Automatic replica set or sharded cluster discovery.
- Update flashback without `fullDocumentBeforeChange`.
- Delete flashback without full deleted document data.

CI evidence:

- The `mongo` job generates a real fixture from `mongo:7.0`.
- `TestLiveChangeStream` runs against a real `mongo:7.0` replica set.
- `TestParseLineRejectsMalformedInput` covers malformed JSON and invalid
  `updateDescription` values.
- `FuzzParseLine` and `BenchmarkParseLine` run as CI smoke gates.

### Redis Family

Supported:

- Redis AOF RESP array commands.
- Live Redis PSYNC replication streams through `dblog.WithDSN`.
- Lowercase normalized command names and native typed command events.
- Command plugins for Redis-compatible products and module commands.
- Checkpoint resume when opened through the root registry.
- Safe flashback commands for `LPUSH`, `RPUSH`, `INCR`, `DECR`, `INCRBY`, and
  `DECRBY`-family operations.
- RESP parser fuzz smoke and command parser benchmark smoke gates.

Compatibility behavior:

- Offline parsing accepts RESP array command frames only.
- RDB preambles and mixed RDB/AOF streams are rejected by the offline parser.
- The live PSYNC reader skips the initial Redis RDB snapshot payload before
  command frames.
- Invalid lengths, LF-only frames, oversized arrays, and oversized bulk strings
  are rejected.

Unsupported:

- Redis Cluster or Sentinel discovery.
- TLS-specific DSN handling.
- Offline RDB snapshot parsing.
- Flashback for state-dependent commands such as `SET`, `HSET`, `SADD`, `DEL`,
  or commands that need previous values, TTLs, or membership state.

CI evidence:

- The `redis` job generates a real fixture from `redis:7.2`.
- `TestLiveReplicationStream` runs against a real `redis:7.2` server.
- `TestParseCommandRejectsInvalidRESP` covers malformed RESP limits and RDB or
  mixed stream prefixes.
- `TestLiveDecoderSkipsSizedRDB` covers live PSYNC RDB snapshot skipping.
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
| Malformed input tests | Done | Done | Done | Done |
| Unsupported input tests | Done | Done | Done | Done |
| Fuzz smoke gate | Done | Done | Done | Done |
| Benchmark smoke gate | Done | Done | Done | Done |
| Static gates: lint, vet, vulnerability scan | Done | Done | Done | Done |

## Next Work

### `v0.3.0` Recovery Workflows

- Expand flashback only for operations with enough prior state in the source log.
- Keep lossy or state-dependent reverse operations omitted unless an explicit
  opt-in API is added.
- Keep checkpoint state portable across process restarts.

### `v0.4.0` Operations

- Publish the tested database/log version matrix with each release.
- Keep parser benchmark smoke gates in CI and record release-time baselines.
- Keep `govulncheck`, race tests, lint, and vet required for every backend.

### `v1.0.0` API Stability

- Freeze public root API and backend contracts.
- Document compatibility, deprecation, and module-versioning policy.
- Define supported extension surfaces for product plugins.

## Versioning Rules

- Root module tags use `vX.Y.Z`.
- Backend module tags use module-prefixed tags: `mysql/vX.Y.Z`,
  `postgres/vX.Y.Z`, `mongo/vX.Y.Z`, and `redis/vX.Y.Z`.
- Backend modules track the root module version for `v0.x` tags.
- GitHub Releases and git tags are the public release record.
- Git history is the detailed change log; this repository does not maintain
  separate release notes or changelog files.
