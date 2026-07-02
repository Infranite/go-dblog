# MongoDB-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/mongo.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/mongo)

该 module 是 `go-dblog` 的 MongoDB 族 backend。它解析 MongoDB oplog exports 或
change stream captures 的 newline-delimited JSON records，从 replica set 读取 live
collection change events，并把 MongoDB-specific fields 保留在 typed events 中。

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

## 文档

| 主题 | English | 中文 |
|---|---|---|
| 功能、范围、包结构、live stream、闪回和插件 | [doc/FEATURES.md](./doc/FEATURES.md) | [doc/FEATURES.zh-CN.md](./doc/FEATURES.zh-CN.md) |
| Oplog、change stream、live reader、插件和闪回示例 | [doc/EXAMPLES.md](./doc/EXAMPLES.md) | [doc/EXAMPLES.zh-CN.md](./doc/EXAMPLES.zh-CN.md) |
| 项目 roadmap 和 release scope | [../doc/ROADMAP.md](../doc/ROADMAP.md#mongodb-family) | [../doc/ROADMAP.zh-CN.md](../doc/ROADMAP.zh-CN.md#mongodb-族) |
| 开发和贡献流程 | [../doc/DEVELOPMENT.md](../doc/DEVELOPMENT.md) | [../doc/DEVELOPMENT.zh-CN.md](../doc/DEVELOPMENT.zh-CN.md) |

## License

Apache License 2.0. See [LICENSE](../LICENSE).
