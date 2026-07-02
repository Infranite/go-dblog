# PostgreSQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/postgres.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/postgres)

该 module 是 `go-dblog` 的 PostgreSQL 族 backend。它解析 logical decoding 文本记录，
通过 SQL slot polling 或 PostgreSQL replication protocol 读取 live `test_decoding`
输出，并用 typed events 暴露 transaction 和 row changes。

[English](./README.md)

需要多数据源编排时使用根 [`go-dblog`](../README.md) module。只需要 PostgreSQL 族
logical decoding 文本解析时可直接使用本 module。

## 安装

当前 release：

```bash
go get github.com/Infranite/go-dblog/postgres@v0.4.0
```

该 module 的仓库 tag 是 `postgres/v0.4.0`；调用方使用上面的 semantic version query。

要求：

- Go 1.25 或更新版本。
- `BEGIN`、`COMMIT`、`table schema.table: INSERT: col[type]:value` 这类
  logical decoding 文本记录。
- live reader 需要 PostgreSQL DSN 和 `test_decoding` logical slot name。DSN 增加
  `replication=database` 时使用 wire-level replication protocol。

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres"
)

func main() {
	var registry dblog.Registry
	if err := postgres.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(postgres.Driver,
		dblog.WithSource(dblog.Source{Name: "slot"}),
		dblog.WithReader(strings.NewReader("table public.users: INSERT: id[integer]:1 name[text]:'Ada'\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		change := event.Body().(postgres.Change)
		fmt.Println(event.Kind(), change.Schema, change.Table, len(change.Columns))
	}
}
```

## 文档

| 主题 | English | 中文 |
|---|---|---|
| 功能、范围、包结构、live reader、闪回和插件 | [doc/FEATURES.md](./doc/FEATURES.md) | [doc/FEATURES.zh-CN.md](./doc/FEATURES.zh-CN.md) |
| 离线、live reader、插件和闪回示例 | [doc/EXAMPLES.md](./doc/EXAMPLES.md) | [doc/EXAMPLES.zh-CN.md](./doc/EXAMPLES.zh-CN.md) |
| 项目 roadmap 和 release scope | [../doc/ROADMAP.md](../doc/ROADMAP.md#postgresql-family) | [../doc/ROADMAP.zh-CN.md](../doc/ROADMAP.zh-CN.md#postgresql-族) |
| 开发和贡献流程 | [../doc/DEVELOPMENT.md](../doc/DEVELOPMENT.md) | [../doc/DEVELOPMENT.zh-CN.md](../doc/DEVELOPMENT.zh-CN.md) |

## License

Apache License 2.0. See [LICENSE](../LICENSE).
