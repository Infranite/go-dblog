# MySQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/mysql.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/mysql)

该 module 是 `go-dblog` 的 MySQL 族 backend。它解析 MySQL binary log 文件，读取
MySQL replication stream，并把 MySQL、MariaDB、MySQL-compatible 方言细节保留在
backend 原生 typed events 中。

[English](./README.md)

需要多数据源编排时使用根 [`go-dblog`](../README.md) module。只需要 MySQL 族
binlog 解析或 replication stream 读取时可直接使用本 module。

## 安装

当前 release：

```bash
go get github.com/Infranite/go-dblog/mysql@v1.0.0
```

该 module 的仓库 tag 是 `mysql/v1.0.0`；调用方使用上面的 semantic version query。

要求：

- Go 1.25 或更新版本。
- MySQL-family binary log 文件，或启用了 binary logging 且允许读取 replication
  stream 的 MySQL server。

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/Infranite/go-dblog/mysql/common"
	"github.com/Infranite/go-dblog/mysql/decode/decoder"
)

func main() {
	fileDecoder, err := decoder.NewBinFileDecoder("./testdata/mysql-bin.000004")
	if err != nil {
		panic(err)
	}
	defer fileDecoder.Close()

	for event, err := range fileDecoder.Events() {
		if err != nil {
			panic(err)
		}
		fmt.Println(common.EventTypeName(event.Header.EventType), event.Header.LogPos)
	}
}
```

## 文档

| 主题 | English | 中文 |
|---|---|---|
| 功能、范围、包结构、事件支持和插件 | [doc/FEATURES.md](./doc/FEATURES.md) | [doc/FEATURES.zh-CN.md](./doc/FEATURES.zh-CN.md) |
| 离线、live reader、过滤、插件和闪回示例 | [doc/EXAMPLES.md](./doc/EXAMPLES.md) | [doc/EXAMPLES.zh-CN.md](./doc/EXAMPLES.zh-CN.md) |
| 项目 roadmap 和 release scope | [../doc/ROADMAP.md](../doc/ROADMAP.md#mysql-family) | [../doc/ROADMAP.zh-CN.md](../doc/ROADMAP.zh-CN.md#mysql-族) |
| 开发和贡献流程 | [../doc/DEVELOPMENT.md](../doc/DEVELOPMENT.md) | [../doc/DEVELOPMENT.zh-CN.md](../doc/DEVELOPMENT.zh-CN.md) |

## License

Apache License 2.0. See [LICENSE](../LICENSE).
