# Redis-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/redis.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/redis)

该 module 是 `go-dblog` 的 Redis 族 backend。它解析 Redis AOF RESP array commands，
并能读取 live Redis replication commands。

[English](./README.md)

需要多数据源编排时使用根 [`go-dblog`](../README.md) module。只需要 Redis-family AOF
RESP parsing 时可直接使用本 module。

## 安装

当前还没有发布公开 tag。首个 `v0.1.0` tag 集合发布后：

```bash
go get github.com/Infranite/go-dblog/redis@v0.1.0
```

该 module 的仓库 tag 是 `redis/v0.1.0`；调用方使用上面的 semantic version query。

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

## 包结构

| Package | 用途 |
|---|---|
| `github.com/Infranite/go-dblog/redis` | 常用 import 的 compatibility facade。 |
| `github.com/Infranite/go-dblog/redis/backend` | 显式注册到 `dblog.Registry`。 |
| `github.com/Infranite/go-dblog/redis/decode/decoder` | 原生 streaming decoder、RESP parser 和 plugin options。 |
| `github.com/Infranite/go-dblog/redis/decode/events/types` | 原生命令、event 和 plugin types。 |

## 已支持

- Redis AOF records 的 RESP array command parsing。
- 通过 `dblog.WithDSN` 打开 live replication stream。
- 小写归一化 command name。
- Streaming RESP decoder。
- 通过 `redis/backend` 集成根 registry。
- 通过根 registry 打开时支持 `dblog.WithCheckpoint`。
- 对无需读取 Redis state 即可安全反转的操作生成闪回命令。
- 面向 Redis-compatible 产品和 module commands 的 command plugin。

## 暂不支持

- Redis Cluster 或 Sentinel discovery。
- TLS-specific DSN 处理。
- RDB snapshot parsing。
- 依赖旧值、TTL、set/hash membership state 的命令闪回，例如 `SET`、`HSET`、`SADD`、
  `DEL`。

## 支持输入

| 输入 | 状态 | CI 证据 |
|---|---|---|
| Redis AOF RESP array commands | 支持 | `redis` fixture job 从 `redis:7.2` 生成；`FuzzParseCommand` smoke target。 |
| Redis replication streams | 支持 | `redis` CI job 启动 `redis:7.2`，写入 SET/INCR/LPUSH，并通过 `dblog.WithDSN` 加 `dblog.WithContext` 读取。 |
| LF-only line endings、empty command names、invalid lengths、oversized arrays/bulk strings | 拒绝 | Parser tests 和 fuzz smoke target。 |
| 最多 8,192 个 RESP array elements 和 8 MiB per bulk string 的 commands | 支持 | Parser limits 由 fuzz smoke 覆盖。 |

## 闪回范围

| Command | 闪回输出 |
|---|---|
| `LPUSH key value ...` | `LPOP key count` |
| `RPUSH key value ...` | `RPOP key count` |
| `INCR`、`DECR`、`INCRBY`、`DECRBY` | 相反的 increment command |

需要 Redis 先前 state、TTL、overwritten value 或成员是否已存在的信息时，不输出闪回。

## 插件

使用 `decoder.WithCommandPlugins` 在事件输出前归一化 Redis module commands 或
Redis-compatible dialects。示例见 [English README](./README.md#command-plugins)。

## 开发

```bash
cd redis && GOWORK=off go test ./...
make integration-redis
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
