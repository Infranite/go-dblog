# PostgreSQL-family Features And Scope

[中文](./FEATURES.zh-CN.md)

This document is the detailed feature and support reference for the
PostgreSQL-family backend. The backend driver name is `pg`, while the module
path remains `postgres`.

## Packages

| Package | Purpose |
|---|---|
| `github.com/Infranite/go-dblog/postgres` | Compatibility facade for common imports. |
| `github.com/Infranite/go-dblog/postgres/backend` | Explicit registration with `dblog.Registry`. |
| `github.com/Infranite/go-dblog/postgres/decode/decoder` | Native streaming decoder, line parser, and plugin options. |
| `github.com/Infranite/go-dblog/postgres/decode/events/types` | Native transaction, change, event, and plugin types. |

## Supported

- Transaction records: `BEGIN` and `COMMIT`.
- Row changes in PostgreSQL logical decoding text form.
- Scalar parsing for `null`, booleans, integers, floats, and quoted strings.
- Streaming line decoder with bounded scanner buffers.
- Live SQL logical slot reader for PostgreSQL `test_decoding` output.
- Wire-level logical replication reader for PostgreSQL `test_decoding` output.
- Root registry integration through `postgres/backend`.
- Checkpoint resume through `dblog.WithCheckpoint` when opened through the root
  registry.
- SQL flashbacks for inserts, deletes, and updates with complete old/new tuple
  data.
- `dblog.RecoveryPlan` steps that pair reverse SQL with source checkpoints.
- Event plugins for PostgreSQL-compatible sources with extra line types.

## Unsupported

- `pgoutput` binary relation and tuple messages.
- Raw WAL or page decoding.
- Output plugins outside `test_decoding` unless a custom text event plugin
  handles them.
- Update flashback when the old tuple does not cover every new tuple column.

## Supported Inputs

| Input | Status | CI evidence |
|---|---|---|
| `BEGIN` and `COMMIT` records from logical decoding text output | Supported | Unit tests and PostgreSQL fixture job generated from `postgres:16`. |
| Row changes in `table schema.table: OPERATION: col[type]:value` form | Supported | Unit tests, fixture job, and `FuzzParseLine` smoke target. |
| `UPDATE: old-key: ... new-tuple: ...` records with complete old tuple data | Supported | Unit tests, fuzz seed, and PostgreSQL fixture job with `REPLICA IDENTITY FULL`. |
| Recovery plan steps for reverse SQL and checkpoint handoff | Supported | `Example_recoveryPlan`. |
| Live SQL logical slot polling with `test_decoding` | Supported | `TestLiveLogicalDecoding` runs against a real `postgres:16` container in CI. |
| Wire-level logical replication with `test_decoding` | Supported | `TestWireLogicalReplication` runs against a real `postgres:16` container in CI. |
| Empty table or operation names | Rejected | Parser tests and fuzz smoke target. |
| `pgoutput` binary relation/tuple messages | Unsupported and rejected by the text parser | `TestParseLineRejectsPgoutputBinaryMessages`. |

## Live Readers

The SQL reader polls `pg_logical_slot_get_changes` and parses the returned
`test_decoding` text with the same parser as offline records.

The wire reader is selected by adding `replication=database` to the DSN. It sends
`START_REPLICATION` for the configured slot, reads CopyData messages, and parses
the embedded `test_decoding` text with the same parser.

Both live readers are intentionally text-oriented in `v1.0.0`. Use
`test_decoding` or a custom text output plugin that can be normalized through
`decoder.WithEventPlugins`; binary `pgoutput` relation and tuple messages are
not decoded by this backend.

## Flashback Scope

| Event | Flashback output |
|---|---|
| `insert` | `DELETE FROM ... WHERE ...;` |
| `update` with complete `old-key` and `new-tuple` columns | `UPDATE ... SET old_values WHERE new_values;` |
| `delete` | `INSERT INTO ... VALUES ...;` |
| `update` without complete old/new tuple data, `begin`, `commit` | No flashback output. |

Update flashback needs enough old values to restore every new tuple column. If
the old tuple only contains key columns, the backend leaves the event out.
`dblog.RecoveryPlan` emits the same reverse SQL plus the checkpoint of the
original event.

## Plugin Support

Use `decoder.WithEventPlugins` to handle line families outside the built-in
logical decoding text records. Plugins receive the original line after the
built-in parser declines it. Plugin output should remain in the backend-native
`types.Event` shape so the root adapter can preserve source, position, and
checkpoint behavior.
