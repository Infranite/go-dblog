# 恢复 Cookbook

[English](./RECOVERY.md)

`go-dblog` 的恢复能力保持保守：只有 source log 包含足够旧状态时，公共 API 才会输出
安全补偿操作。补偿操作保留 backend-native 类型，replay 代码可以使用对应数据库的原生
写入路径。

## 通用流程

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

`RecoveryPlan` 流式输出 `dblog.RecoveryStep`：

| 字段 | 含义 |
|---|---|
| `Operation` | backend-native 补偿操作，例如反向 SQL、Mongo command、Redis command 或 MySQL row event。 |
| `Checkpoint` | 原始事件的 source 和 position。补偿操作持久 replay 成功后再保存。 |

如果要对一个有界窗口做 point-in-time rollback，先持久化这些 step，再按 checkpoint
倒序 replay。流式补偿场景可以边读边 replay，并在每个 step 成功后保存 checkpoint。

replay worker 重启后，用 `dblog.WithCheckpoint(checkpoint)` 重新打开 decoder。

## 产品要求

| 产品 | 安全恢复输入 |
|---|---|
| MySQL 族 | 具备 table-map metadata 的完整 write、delete、update row image。 |
| PostgreSQL 族 | insert、delete，以及 `old-key` 覆盖所有 `new-tuple` columns 的 update；需要 update 恢复时使用 `REPLICA IDENTITY FULL`。 |
| MongoDB 族 | 带 `documentKey` 的 insert、带 `fullDocumentBeforeChange` 的 update/replace，以及带完整 deleted document data 的 delete。 |
| Redis 族 | list push、numeric increment 等确定性 command。依赖状态的 command 会被省略。 |

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

每个 backend 的 examples 文档都包含产品专属恢复片段：

| 产品 | 示例 |
|---|---|
| MySQL 族 | [mysql/doc/EXAMPLES.zh-CN.md](../mysql/doc/EXAMPLES.zh-CN.md#构建恢复计划) |
| PostgreSQL 族 | [postgres/doc/EXAMPLES.zh-CN.md](../postgres/doc/EXAMPLES.zh-CN.md#构建带-checkpoint-的恢复计划) |
| MongoDB 族 | [mongo/doc/EXAMPLES.zh-CN.md](../mongo/doc/EXAMPLES.zh-CN.md#基于-pre-images-构建恢复计划) |
| Redis 族 | [redis/doc/EXAMPLES.zh-CN.md](../redis/doc/EXAMPLES.zh-CN.md#构建带-checkpoint-的恢复计划) |
