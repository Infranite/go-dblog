# Roadmap

This roadmap tracks user-visible capability, release readiness, and engineering
gates for `go-dblog`. It is not a commitment to dates.

## Status

| Status | Meaning |
|---|---|
| Done | Implemented, documented, and covered by CI. |
| In progress | Actively being built on the main development branch. |
| Planned | Accepted scope, not yet started. |
| Candidate | Useful direction, still needs design or user validation. |
| Deferred | Explicitly out of the current release line. |

## Release Line

| Release | Status | Theme | Deliverables | Exit gates |
|---|---|---|---|---|
| `v0.1.0` | In progress | Offline parser developer preview | Root common API, MySQL binlog file parser, PostgreSQL logical decoding text parser, MongoDB JSON line parser, Redis RESP AOF parser, plugin hooks, filtering, basic flashback helpers. | Protected PR `ci` check passes: lint, vet, vulnerability scan, unit tests, and real fixture-backed MySQL, MongoDB, PostgreSQL, and Redis integration tests. README documents offline scope and module tags. |
| `v0.2.0` | Planned | Compatibility hardening | Compatibility fixtures and negative cases for each backend; documented supported inputs and known gaps per backend. | Backend README files include supported versions/formats, unsupported cases, fixture source, and parser behavior for unknown events. |
| `v0.3.0` | Planned | Live readers | MySQL replication reader, PostgreSQL logical replication reader, MongoDB change stream reader, Redis replication stream reader. | Live readers implement `dblog.Decoder`, support context cancellation, and have integration tests isolated from unit tests. |
| `v0.4.0` | Planned | Recovery workflows | Checkpoint model, resumable decoding hooks, expanded flashback operations, and unsafe-operation guardrails. | Recovery APIs are backend-neutral; lossy or state-dependent reverse operations are documented and opt-in. |
| `v0.5.0` | Planned | Operational maturity | Benchmark baselines, compatibility matrix, fuzz targets for parsers, release notes with tested versions. | CI runs fuzz smoke tests, benchmark smoke tests, and publishes tested backend/version matrix. |
| `v1.0.0` | Candidate | Stable public API | Frozen root API, stable backend package contracts, migration notes from `v0.x`. | No known API blockers; compatibility policy and deprecation policy are documented. |

## Capability Matrix

| Capability | MySQL | PostgreSQL | MongoDB | Redis |
|---|---|---|---|---|
| Offline parser | Done | Done | Done | Done |
| Native typed events | Done | Done | Done | Done |
| Common `dblog.Event` adapter | Done | Done | Done | Done |
| Plugin hooks | Done | Done | Done | Done |
| Basic filtering | Done | Done | Done | Done |
| Basic flashback | Partial | Partial | Partial | Partial |
| Compatibility matrix | Planned | Planned | Planned | Planned |
| Live reader | Planned | Planned | Planned | Planned |
| Checkpoint/resume | Planned | Planned | Planned | Planned |
| Fuzz coverage | Planned | Planned | Planned | Planned |
| Throughput baseline | Planned | Planned | Planned | Planned |

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

- define checkpoint and resume contracts;
- expand safe flashback output per backend;
- reject or omit reverse operations that require missing prior state;
- document lossy cases at the backend level.

Done when:

- flashback behavior is tested per supported operation;
- unsafe reverse operations are not emitted silently;
- checkpoint state is portable across process restarts.

### 4. Operations

Goal: make releases measurable and safe to adopt.

Required work:

- add parser benchmarks for representative inputs;
- add fuzz smoke targets for trust-boundary parsers;
- publish tested backend/version matrix;
- run vulnerability scanning in CI.

Done when:

- release notes include tested database/log versions;
- benchmark smoke checks run in CI;
- `govulncheck` and race tests are required checks.

## Versioning Rules

- Root module tags use `vX.Y.Z`.
- Backend module tags use module-prefixed tags: `mysql/vX.Y.Z`,
  `mongo/vX.Y.Z`, `postgres/vX.Y.Z`, and `redis/vX.Y.Z`.
- Backend modules track the root module version for `v0.x` releases.
- Breaking API changes are allowed before `v1.0.0`, but must be documented in
  `CHANGELOG.md`.

## Maintenance Rules

- Move an item to Done only when its exit gates are satisfied in CI or release
  evidence.
- Add new work to an existing workstream before creating a new release line.
- Keep release scope user-visible; internal refactors belong in issues or PRs,
  not the roadmap.
- Update `CHANGELOG.md` when a roadmap item changes shipped behavior.

## Non-Goals Before `v1.0.0`

- GUI tools.
- Managed service connectors.
- Cross-database semantic normalization beyond the common `dblog.Event` shape.
- Flashback output for operations that need state missing from the source log.
