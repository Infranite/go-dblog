# MongoDB-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/mongo.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/mongo)

该 module 是 `go-dblog` 的 MongoDB 族 backend。它解析 MongoDB oplog exports 或
change stream captures 的 newline-delimited JSON records，并把 MongoDB-specific
fields 保留在 typed events 中。它也能从 MongoDB replica set 读取 live collection
change events。

[English](./README.md)

需要多数据源编排时使用根 [`go-dblog`](../README.md) module。只需要 MongoDB 族日志
解析时可直接使用本 module。

## 安装

当前 release：

```bash
go get github.com/Infranite/go-dblog/mongo@v0.2.0
```

该 module 的仓库 tag 是 `mongo/v0.2.0`；调用方使用上面的 semantic version query。

要求：

- Go 1.25 或更新版本。
- 来自 oplog export 或 change stream capture 的 one JSON record per line。
- 通过 DSN 打开 live change stream 时需要 MongoDB replica set。

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo"
)

func main() {
	var registry dblog.Registry
	if err := mongo.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(mongo.Driver,
		dblog.WithReader(strings.NewReader(`{"op":"i","ns":"app.users","o":{"_id":1,"name":"Ada"}}`+"\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		change := event.Body().(mongo.Change)
		fmt.Println(event.Kind(), change.Database, change.Collection)
	}
}
```

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
- 输入包含足够 document 或 key 数据时，为 insert、delete、update 生成闪回命令。
- 面向 MongoDB-compatible 产品不同 operation names 或 metadata 的 event plugin。

## 暂不支持

- JSON records 或 change streams 之外的 raw oplog tailing。
- 自动 replica set 或 sharded cluster discovery。
- 缺少 `fullDocumentBeforeChange` 的 update 闪回。
- 缺少完整 deleted document data 的 delete 闪回。

## 支持输入

| 输入 | 状态 | CI 证据 |
|---|---|---|
| 带 `op`、`ns`、`o`、`o2` 的 MongoDB oplog JSON records | 支持 | `mongo` fixture job 从 `mongo:7.0` 生成；`FuzzParseLine` smoke target。 |
| 带 change stream fields 的 MongoDB change stream JSON records | 支持 | Unit tests 和 `FuzzParseLine` seeds 覆盖有效和 malformed records。 |
| 来自 MongoDB replica set 的 live collection change streams | 支持 | `mongo` CI job 启动 `mongo:7.0`，写入 insert/update/delete，并通过 `dblog.WithDSN` 加 `dblog.WithContext` 读取。 |
| malformed JSON records 或非 object 的 `updateDescription` | 拒绝 | `TestParseLineRejectsMalformedInput` 和 `FuzzParseLine`。 |
| empty operation names | 拒绝 | Parser tests 和 fuzz smoke target。 |
| unknown non-empty operation names | 作为 backend event kinds 输出，除非 decoder plugin 归一化 | Plugin tests 和 parser tests。 |

## Live Change Streams

使用 `dblog.WithDSN` 和 `db.collection` 形式的 source name 打开 live reader。
MongoDB 必须以 replica set 运行，因为 standalone server 不支持 change streams。

live change stream 的 update 闪回需要 `fullDocumentBeforeChange`。如果业务需要反向
update command，需要在源 collection 上启用 change stream pre-images。没有 pre-image
时 update event 仍会解码，但 `dblog.Flashbacks` 不输出反向命令。

## 闪回范围

| Event | 闪回输出 |
|---|---|
| 带 `documentKey` 的 `insert` | `mongo.Command{Operation: "delete", Filter: documentKey}` |
| 带 `documentKey` 和 `fullDocumentBeforeChange` 的 `update` | `mongo.Command{Operation: "replace", Filter: documentKey, Document: fullDocumentBeforeChange}` |
| 带 full document data 的 `delete` | `mongo.Command{Operation: "insert", Document: document}` |
| 缺少 before-image 的 `update`、`command`、`noop` | 不输出闪回。 |

malformed JSON input 和非 object 的 `updateDescription` 会在事件输出前被拒绝。

## 插件

使用 `decoder.WithEventPlugins` 在输出前归一化 MongoDB-compatible source 的 event
shape。示例见 [English README](./README.md#event-plugins)。

## 开发

```bash
cd mongo && GOWORK=off go test ./...
make integration-mongo
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
