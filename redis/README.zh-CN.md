# Redis-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/redis.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/redis)

该 module 是 `go-dblog` 的 Redis 族 backend。它解析 Redis AOF RESP array commands，
并能读取 live Redis replication commands。

[English](./README.md)

需要多数据源编排时使用根 [`go-dblog`](../README.md) module。只需要 Redis-family AOF
RESP parsing 时可直接使用本 module。

## 安装

当前 release：

```bash
go get github.com/Infranite/go-dblog/redis@v0.3.0
```

该 module 的仓库 tag 是 `redis/v0.3.0`；调用方使用上面的 semantic version query。

要求：

- Go 1.25 或更新版本。
- Redis AOF command frames encoded as RESP arrays，或通过 TCP 可访问的 Redis server。

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis"
)

func main() {
	var registry dblog.Registry
	if err := redis.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(redis.Driver,
		dblog.WithSource(dblog.Source{Name: "appendonly.aof"}),
		dblog.WithReader(strings.NewReader("*3\r\n$4\r\nSADD\r\n$4\r\ntags\r\n$2\r\ngo\r\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		command := event.Body().(redis.Command)
		fmt.Println(event.Kind(), command.Args)
	}
}
```

## 文档

| 主题 | English | 中文 |
|---|---|---|
| 功能、范围、包结构、RDB 行为、闪回和插件 | [doc/FEATURES.md](./doc/FEATURES.md) | [doc/FEATURES.zh-CN.md](./doc/FEATURES.zh-CN.md) |
| AOF、live reader、插件和闪回示例 | [doc/EXAMPLES.md](./doc/EXAMPLES.md) | [doc/EXAMPLES.zh-CN.md](./doc/EXAMPLES.zh-CN.md) |
| 项目 roadmap 和 release scope | [../doc/ROADMAP.md](../doc/ROADMAP.md#redis-family) | [../doc/ROADMAP.zh-CN.md](../doc/ROADMAP.zh-CN.md#redis-族) |
| 开发和贡献流程 | [../doc/DEVELOPMENT.md](../doc/DEVELOPMENT.md) | [../doc/DEVELOPMENT.zh-CN.md](../doc/DEVELOPMENT.zh-CN.md) |

## License

Apache License 2.0. See [LICENSE](../LICENSE).
