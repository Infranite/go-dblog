# Roadmap

This roadmap tracks user-visible capability, version scope, and engineering
gates for `go-dblog`. It is not a commitment to dates.

## Status

| Status | Meaning |
|---|---|
| Done | Implemented, documented, and covered by CI. |
| Ready | Implemented and covered by CI; ready for a public tag. |
| Partial | A safe documented subset is implemented and covered by CI. |
| In progress | Actively being built on the main development branch. |
| Planned | Accepted scope, not yet started. |
| Candidate | Useful direction, still needs design or user validation. |
| Unsupported | Explicitly not emitted or accepted in this version line. |
| Deferred | Explicitly out of the current version line. |

## Version Targets

GitHub Releases and Git tags are the public release record. Git history remains
the detailed change log; this file only tracks user-visible scope and exit
gates.

| Version | Status | Theme | Deliverables | Exit gates |
|---|---|---|---|---|
| `v0.1.0` | In progress | Parser and CDC developer preview | Root common API, MySQL binlog file parser, PostgreSQL logical decoding text parser and SQL slot reader, MongoDB JSON parser plus live change stream reader, Redis RESP AOF parser plus PSYNC replication stream reader, plugin hooks, filtering, checkpoint resume, and safe flashback helpers where the log contains enough data. | Protected PR `ci` and `merge-policy` checks pass: lint, vet, vulnerability scan, unit tests, and real fixture-backed MySQL, MongoDB, PostgreSQL, and Redis integration tests. README documents shipped scope and module tags. |
| `v0.2.0` | Planned | Compatibility hardening | Compatibility fixtures and negative cases for each backend; documented supported inputs and known gaps per backend. | Backend README files include supported versions/formats, unsupported cases, fixture source, and parser behavior for unknown events. |
| `v0.3.0` | Planned | Remaining live readers | MySQL replication reader and PostgreSQL wire-level logical replication protocol reader. | Live readers implement `dblog.Decoder`, support context cancellation, and have integration tests isolated from unit tests. |
| `v0.4.0` | Planned | Recovery workflows | Expanded flashback operations and unsafe-operation guardrails. | Recovery APIs are backend-neutral; lossy or state-dependent reverse operations are documented and opt-in. |
| `v0.5.0` | Planned | Operational maturity | Tested backend/version matrix and long-running benchmark history. | CI runs fuzz smoke tests, benchmark smoke tests, and publishes the tested backend/version matrix. |
| `v1.0.0` | Candidate | Stable public API | Frozen root API, stable backend package contracts, migration notes from `v0.x`. | No known API blockers; compatibility policy and deprecation policy are documented. |

## Capability Matrix

Rows marked Done, Partial, or Unsupported are protected by the `ci` workflow.
The final `ci` job requires every referenced job below to pass on pull requests,
merge queue runs, and `master` pushes.

| Capability | MySQL | PostgreSQL | MongoDB | Redis | CI evidence |
|---|---|---|---|---|---|
| Offline parser | Done: local MySQL-family binlog files | Done: logical decoding text records | Done: JSONL oplog and change stream records | Done: RESP array AOF commands | `mysql`, `postgres`, `mongo`, and `redis` jobs generate real fixtures and run `go test -race -count=1 -shuffle=on ./...`. |
| Native typed events | Done | Done | Done | Done | Backend package tests assert typed bodies and event fields; fixture jobs exercise the real decoders. |
| Common `dblog.Event` adapter | Done | Done | Done | Done | `root_test`, backend registration tests, and MySQL `TestDblogDecoderEvents`. |
| Plugin hooks | Done: event plugins plus built-in MariaDB plugin | Done: event plugins | Done: event plugins | Done: command plugins | `mysql/plugin/mariadb`, `postgres/decode/decoder`, `mongo/decode/decoder`, and `redis/decode/decoder` plugin tests. |
| Basic filtering | Done | Done | Done | Done | `dblog.TestFilterAppliesPredicates`; fixture-backed backend tests filter real decoded events. |
| Safe flashback | Partial: typed reverse row events for complete write/delete/update row images | Partial: insert/delete SQL plus update SQL with complete old/new tuple data | Partial: insert/delete/update-with-before-image commands | Partial: LPUSH/RPUSH and INCR-family commands | `dblog.TestFlashbacksYieldsReverseOperations`; fixture-backed backend tests assert emitted operations; `mysql/decode/decoder.TestDblogEventReverseRowsEvents` asserts MySQL row flashback output and `TestDblogEventReverseSkipsUnsafeRowsEvents` rejects incomplete row images; `postgres.TestParseLineFlashbackRestoresUpdateOldKey` asserts complete old tuple update restore and PostgreSQL fixture CI uses `REPLICA IDENTITY FULL`; `mongo.TestParseLineFlashbackRestoresUpdateBeforeImage` asserts update restore uses `fullDocumentBeforeChange`; Redis fixture CI rejects state-dependent HDEL/SREM flashback output. |
| Fixture provenance | Done: generated from MySQL 5.6, 5.7, 8.0, and 8.4 containers | Done: generated from PostgreSQL 16 | Done: generated from MongoDB 7.0 | Done: generated from Redis 7.2 | Workflow fixture generation steps run before integration tests. |
| Static quality gates | Done | Done | Done | Done | `lint`, `vet`, and `vuln` matrix jobs run with `GOWORK=off` for every module. |
| Compatibility matrix | Done | Done | Done | Done | Backend README files publish supported inputs, limits, and tested fixture versions; CI fixture jobs and fuzz seeds prove those claims. |
| Live reader | Planned for `v0.3.0` | Partial: SQL logical slot polling for `test_decoding` output | Done: collection change streams from replica sets | Done: PSYNC replication stream | `postgres.TestLiveLogicalDecoding` starts a real `postgres:16` container, creates a logical slot, writes rows, and reads them through `dblog.WithDSN` plus `dblog.WithContext`; `mongo.TestLiveChangeStream` starts a real `mongo:7.0` replica set, writes insert/update/delete operations, and reads them through `dblog.WithDSN` plus `dblog.WithContext`; `redis.TestLiveReplicationStream` starts a real `redis:7.2`, writes SET/INCR/LPUSH, and reads them through `dblog.WithDSN` plus `dblog.WithContext`; PostgreSQL wire-level replication protocol remains planned. |
| Checkpoint/resume | Done | Done | Done | Done | `dblog.TestCheckpointOfAndOpenOptions` and backend `TestRegisterResumesAfterCheckpoint` tests prove `dblog.WithCheckpoint` resumes after the stored event position; fixture-backed CI covers MySQL binlog checkpoints with generated binlogs. |
| Fuzz coverage | Done: event header parser | Done: logical decoding line parser | Done: JSON line parser | Done: RESP command parser | `fuzz` matrix job runs parser fuzz smoke targets for every backend. |
| Throughput baseline | Done: fixture decoder benchmark | Done: line parser benchmark | Done: line parser benchmark | Done: RESP command benchmark | `bench` matrix job runs parser benchmark smoke targets for every backend. |

## Workstreams

### 1. Parser Compatibility

Goal: make each parser predictable across supported input variants.

Required work:

- add fixture provenance for every backend;
- cover malformed input, unknown events, and partial records;
- document how unknown or unsupported records are represented;
- keep backend-specific parsing details inside backend modules.

Done when:

- each backend README has a supported-input table;
- parser tests include happy path, malformed input, unsupported input, and
  compatibility behavior;
- CI proves each backend works with `GOWORK=off`.

### 2. Live Readers

Goal: support production-style CDC ingestion without changing the common API.

Required work:

- add context-aware readers for online sources;
- keep file/offline decoders as the simplest supported path;
- isolate integration tests from unit tests;
- document connection requirements and failure behavior.

Done when:

- every live reader implements `dblog.Decoder`;
- cancellation and `Close` behavior are tested;
- integration tests can run with explicit opt-in services.

### 3. Recovery

Goal: provide recovery helpers without inventing data that is not in the log.

Required work:

- expand safe flashback output per backend;
- reject or omit reverse operations that require missing prior state;
- document lossy cases at the backend level.

Done when:

- flashback behavior is tested per supported operation;
- unsafe reverse operations are not emitted silently;
- checkpoint state remains portable across process restarts.

### 4. Operations

Goal: make published versions measurable and safe to adopt.

Required work:

- add parser benchmarks for representative inputs;
- add fuzz smoke targets for trust-boundary parsers;
- publish tested backend/version matrix;
- run vulnerability scanning in CI.

Done when:

- the tested database/log versions are recorded with the tag or GitHub Release;
- benchmark smoke checks run in CI;
- `govulncheck` and race tests are required checks.

## Versioning Rules

- Root module tags use `vX.Y.Z`.
- Backend module tags use module-prefixed tags: `mysql/vX.Y.Z`,
  `mongo/vX.Y.Z`, `postgres/vX.Y.Z`, and `redis/vX.Y.Z`.
- Backend modules track the root module version for `v0.x` tags.
- Breaking API changes are allowed before `v1.0.0`, but must be visible in the
  GitHub Release entry or commit history when a tag is published.

## Maintenance Rules

- Move an item to Done only when its exit gates are satisfied in CI or tag
  evidence.
- Add new work to an existing workstream before creating a new version line.
- Keep version scope user-visible; internal refactors belong in issues or PRs,
  not the roadmap.
- For published tags, keep GitHub Release entries aligned with shipped behavior.

## Non-Goals Before `v1.0.0`

- GUI tools.
- Managed service connectors.
- Cross-database semantic normalization beyond the common `dblog.Event` shape.
- Flashback output for operations that need state missing from the source log.
