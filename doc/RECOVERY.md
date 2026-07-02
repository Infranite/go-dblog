# Recovery Cookbook

[中文](./RECOVERY.zh-CN.md)

`go-dblog` recovery is intentionally conservative. The common API only emits
safe compensating operations when the source log contains enough prior state.
Backend-specific operation types are preserved so replay code can use the
database driver's native write path.

## Common Flow

```go
for step, err := range dblog.RecoveryPlan(decoder.Events()) {
	if err != nil {
		panic(err)
	}

	if err := replay(step.Operation); err != nil {
		panic(err)
	}
	saveCheckpoint(step.Checkpoint)
}
```

`RecoveryPlan` streams `dblog.RecoveryStep` values. Each step contains:

| Field | Meaning |
|---|---|
| `Operation` | Backend-native compensating operation, such as reverse SQL, a Mongo command, a Redis command, or a MySQL row event. |
| `Checkpoint` | Source and position of the original event. Persist it after the compensating operation is durably replayed. |

For point-in-time rollback of a bounded window, persist the emitted steps and
replay them in reverse checkpoint order. For streaming compensation, apply each
step as it is emitted and then persist the checkpoint.

Use `dblog.WithCheckpoint(checkpoint)` when reopening a decoder after a replay
worker restarts.

## Product Requirements

| Product | Safe recovery inputs |
|---|---|
| MySQL family | Complete write, delete, and update row images with table-map metadata. |
| PostgreSQL family | Inserts, deletes, and updates whose `old-key` covers every `new-tuple` column; use `REPLICA IDENTITY FULL` when update recovery is required. |
| MongoDB family | Inserts with `documentKey`, updates/replaces with `fullDocumentBeforeChange`, and deletes with full deleted document data. |
| Redis family | Deterministic commands such as list pushes and numeric increments. State-dependent commands are omitted. |

## Replay Type Switch

```go
switch op := step.Operation.(type) {
case string:
	// PostgreSQL reverse SQL.
case mongo.Command:
	// MongoDB insert/delete/replace command.
case redis.Command:
	// Redis compensating command.
case *events.Event:
	// MySQL-family reverse row event.
}
```

Each backend examples document contains a product-specific recovery snippet:

| Product | Examples |
|---|---|
| MySQL family | [mysql/doc/EXAMPLES.md](../mysql/doc/EXAMPLES.md#build-a-recovery-plan) |
| PostgreSQL family | [postgres/doc/EXAMPLES.md](../postgres/doc/EXAMPLES.md#build-a-recovery-plan-with-checkpoints) |
| MongoDB family | [mongo/doc/EXAMPLES.md](../mongo/doc/EXAMPLES.md#build-a-recovery-plan-with-pre-images) |
| Redis family | [redis/doc/EXAMPLES.md](../redis/doc/EXAMPLES.md#build-a-recovery-plan-with-checkpoints) |
