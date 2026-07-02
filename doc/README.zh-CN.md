# go-dblog

[![CI](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml/badge.svg?branch=master)](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog)
[![Go Report Card](https://goreportcard.com/badge/github.com/Infranite/go-dblog)](https://goreportcard.com/report/github.com/Infranite/go-dblog)
[![License](https://img.shields.io/github/license/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/LICENSE)

`go-dblog` 是一个多模块 Go 工具包，用于解析数据库变更日志。根模块提供统一事件、
backend 注册、checkpoint、过滤和闪回契约；每个产品 backend 保留自己的原生事件模型
和依赖。

[English](../README.md)

## 产品索引

只安装实际使用的 backend。

| 产品 | Module | 详情 |
|---|---|---|
| 公共 API | `github.com/Infranite/go-dblog` | [README](../README.md) |
| MySQL 族 | `github.com/Infranite/go-dblog/mysql` | [English](../mysql/README.md) / [中文](../mysql/README.zh-CN.md) |
| PostgreSQL 族 | `github.com/Infranite/go-dblog/postgres` | [English](../postgres/README.md) / [中文](../postgres/README.zh-CN.md) |
| MongoDB 族 | `github.com/Infranite/go-dblog/mongo` | [English](../mongo/README.md) / [中文](../mongo/README.zh-CN.md) |
| Redis 族 | `github.com/Infranite/go-dblog/redis` | [English](../redis/README.md) / [中文](../redis/README.zh-CN.md) |

当前已支持和暂不支持的数据源细节见 [doc/ROADMAP.md](./ROADMAP.md)。

## 功能

- 面向混合数据库日志流的 backend-neutral `dblog.Event`。
- 通过 `dblog.Registry` 显式注册 backend。
- 基于 Go iterator API 的流式 decoder。
- 公共层提供 source metadata、position、checkpoint resume、过滤和安全闪回辅助。
- 各 backend 保留数据库原生 typed event。
- backend decoder 包内提供插件入口，用于方言记录。
- backend 独立 Go module，调用方不会安装无关数据库依赖。

## 安装

当前公开 tag 集合是 `v0.2.0`。

```bash
go get github.com/Infranite/go-dblog@v0.2.0
go get github.com/Infranite/go-dblog/mysql@v0.2.0
go get github.com/Infranite/go-dblog/postgres@v0.2.0
go get github.com/Infranite/go-dblog/mongo@v0.2.0
go get github.com/Infranite/go-dblog/redis@v0.2.0
```

## 最小示例

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
		dblog.WithReader(strings.NewReader("*2\r\n$4\r\nINCR\r\n$7\r\ncounter\r\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		fmt.Println(event.Kind(), dblog.PositionOf(event).Value)
	}
}
```

需要多数据源路由、共享过滤、CDC pipeline、backend 注册和恢复任务时使用公共 API。
需要数据库特有字段时，直接使用 backend 原生 API。

## 文档

| 主题 | English | 中文 |
|---|---|---|
| 项目总览 | [README](../README.md) | 本文档 |
| Roadmap 和产品范围 | [doc/ROADMAP.md](./ROADMAP.md) | [doc/ROADMAP.zh-CN.md](./ROADMAP.zh-CN.md) |
| 开发和贡献流程 | [doc/DEVELOPMENT.md](./DEVELOPMENT.md) | [doc/DEVELOPMENT.zh-CN.md](./DEVELOPMENT.zh-CN.md) |
| 安全策略 | [doc/SECURITY.md](./SECURITY.md) | [doc/SECURITY.zh-CN.md](./SECURITY.zh-CN.md) |

GitHub Releases 和 git tags 是公开发布记录。Git history 是详细变更记录；项目不单独
维护 release notes 或 changelog 文件。

## License

Apache License 2.0. See [LICENSE](../LICENSE).
