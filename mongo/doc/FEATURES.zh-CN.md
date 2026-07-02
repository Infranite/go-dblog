# MongoDB-family 功能与范围

[English](./FEATURES.md)

本文档是 MongoDB 族 backend 的详细功能和支持范围说明。

## 包结构

| Package | 用途 |
|---|---|
| `github.com/Infranite/go-dblog/mongo` | 常用 import 的 compatibility facade。 |
| `github.com/Infranite/go-dblog/mongo/backend` | 显式注册到 `dblog.Registry`。 |
| `github.com/Infranite/go-dblog/mongo/decode/decoder` | 原生 streaming decoder、line parser 和 plugin options。 |
| `github.com/Infranite/go-dblog/mongo/decode/events/types` | 原生 change、command、event 和 plugin types。 |

## 已支持

- 带 `op`、`ns`、`o`、`o2` fields 的 oplog JSON records。
- 带 `operationType`、`ns`、`documentKey`、`fullDocument`、
  `fullDocumentBeforeChange`、`updateDescription` 的 change stream JSON records。
- 通过 `dblog.WithDSN` 和 `dblog.WithSource(dblog.Source{Name: "db.collection"})`
  打开 live collection change stream。
- 有界 scanner buffer 的 streaming line decoder。
- 通过 `mongo/backend` 集成根 registry。
- 通过根 registry 打开时支持 `dblog.WithCheckpoint`。
- 输入包含足够 document 或 key 数据时，为 insert、delete、update、replace 生成闪回命令。
- `dblog.RecoveryPlan` 会把 flashback command 和 source checkpoint 组成恢复 step。
- 面向 MongoDB-compatible 产品不同 operation names 或 metadata 的 event plugin。

## 暂不支持

- JSON records 或 change streams 之外的 raw oplog tailing。
- 自动 replica set 或 sharded cluster discovery。
- 缺少 `fullDocumentBeforeChange` 的 update 或 replace 闪回。
- 缺少完整 deleted document data 的 delete 闪回。

## 支持输入

| 输入 | 状态 | CI 证据 |
|---|---|---|
| 带 `op`、`ns`、`o`、`o2` 的 MongoDB oplog JSON records | 支持 | `mongo` fixture job 从 `mongo:7.0` 生成；`FuzzParseLine` smoke target。 |
| 带 `operationType`、`ns`、`documentKey`、`fullDocument`、`fullDocumentBeforeChange`、`updateDescription` 的 MongoDB change stream JSON records | 支持 | Unit tests 和 `FuzzParseLine` seeds 覆盖有效和 malformed records。 |
| before-image update 和 replace 的 recovery plan step | 支持 | `Example_recoveryPlan` 和 replace flashback tests。 |
| 来自 MongoDB replica set 的 live collection change streams | 支持 | `mongo` CI job 启动 `mongo:7.0`，写入 insert/update/delete，并通过 `dblog.WithDSN` 加 `dblog.WithContext` 读取。 |
| malformed JSON records 或非 object 的 `updateDescription` | 拒绝 | `TestParseLineRejectsMalformedInput` 和 `FuzzParseLine`。 |
| empty operation names | 拒绝 | Parser tests 和 fuzz smoke target。 |
| unknown non-empty operation names | 作为 backend event kinds 输出，除非 decoder plugin 归一化 | Plugin tests 和 parser tests。 |

## Live Change Streams

使用 `dblog.WithDSN` 和 `db.collection` 形式的 source name 打开 live reader。MongoDB
必须以 replica set 运行，因为 standalone server 不支持 change streams。

live change stream 的 update 闪回需要 `fullDocumentBeforeChange`。如果业务需要反向
update command，需要在源 collection 上启用 change stream pre-images。没有 pre-image
时 update event 仍会解码，但 `dblog.Flashbacks` 不输出反向命令。

## 闪回范围

| Event | 闪回输出 |
|---|---|
| 带 `documentKey` 的 `insert` | `mongo.Command{Operation: "delete", Filter: documentKey}` |
| 带 `documentKey` 和 `fullDocumentBeforeChange` 的 `update` 或 `replace` | `mongo.Command{Operation: "replace", Filter: documentKey, Document: fullDocumentBeforeChange}` |
| 带 full document data 的 `delete` | `mongo.Command{Operation: "insert", Document: document}` |
| 缺少 before-image 的 `update` 或 `replace`、`command`、`noop` | 不输出闪回。 |

Update 和 replace 闪回使用完整 before-image 作为 replacement document。没有
before-image data 的事件不会输出闪回。Malformed JSON input 和非 object 的
`updateDescription` 会在事件输出前被拒绝。
`dblog.RecoveryPlan` 输出相同 command，并附带原始事件 checkpoint。

## 插件支持

使用 `decoder.WithEventPlugins` 在输出前归一化 MongoDB-compatible source 的 event
shape。插件可以重命名 operation、补充产品特有 metadata，或把兼容 event family 映射到
backend-native `types.Change` 形态。
