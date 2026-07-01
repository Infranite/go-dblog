# go-dblog

[![CI](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml/badge.svg?branch=develop)](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/develop/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog)
[![Go Report Card](https://goreportcard.com/badge/github.com/Infranite/go-dblog)](https://goreportcard.com/report/github.com/Infranite/go-dblog)
[![License](https://img.shields.io/github/license/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/develop/LICENSE)

`go-dblog` 是一个面向数据库变更日志解析、CDC 和数据恢复工作流的多模块 Go
工具包。根模块只提供公共事件模型和跨 backend 编排 API；具体数据库族保留自己的
原生事件结构、解析细节和依赖图。

[English](https://github.com/Infranite/go-dblog/blob/develop/README.md)

## 模块

只安装实际使用的 backend。

|模块|范围|状态|
|---|---|---|
|[`github.com/Infranite/go-dblog`](https://pkg.go.dev/github.com/Infranite/go-dblog)|多数据源编排公共 API|已支持|
|[`github.com/Infranite/go-dblog/mysql`](../mysql)|MySQL 族 binlog：MySQL、MariaDB、MySQL-compatible 方言|已支持|
|`github.com/Infranite/go-dblog/postgres`|PostgreSQL 族 logical replication / WAL|计划中|
|`github.com/Infranite/go-dblog/mongo`|MongoDB 族 oplog / change stream|计划中|
|`github.com/Infranite/go-dblog/redis`|Redis 族 AOF / replication stream|计划中|

backend 按数据库族命名，而不是按某一种日志格式命名。MySQL 族使用 `mysql` 模块；
PostgreSQL 族后续使用 `postgres` 模块，以便容纳兼容方言和生态扩展。

## 公共层 API

```bash
go get github.com/Infranite/go-dblog
go get github.com/Infranite/go-dblog/mysql
```

```go
package main

import (
	"fmt"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mysql/decode/decoder"
)

func main() {
	mysqlDecoder, err := decoder.NewDblogDecoder("./testdata/mysql-bin.000004")
	if err != nil {
		panic(err)
	}
	defer mysqlDecoder.Close()

	for event, err := range dblog.Events(mysqlDecoder) {
		if err != nil {
			panic(err)
		}
		source := dblog.SourceOf(event)
		position := dblog.PositionOf(event)
		fmt.Println(source.Driver, position.Value, event.Kind())
	}
}
```

需要数据库完整细节时，直接使用 backend 原生 API。需要多数据源路由、共享过滤、
CDC 或恢复流水线时，再接入根模块公共 API。

## MySQL 族 backend

```bash
go get github.com/Infranite/go-dblog/mysql
```

MySQL 族 backend 支持 MySQL 5.1 及之后版本、MariaDB binlog 扩展，以及
MySQL-compatible 复制协议事件。安装、使用案例、兼容模式、插件机制和事件支持表见
[`mysql/README.md`](../mysql/README.md)。

## 路线图

|阶段|范围|
|---|---|
|1|MySQL 族复制连接 reader|
|2|PostgreSQL 族 logical replication backend|
|3|至少两个真实 backend 稳定后补齐共享 CDC helper|
|4|MongoDB 族 oplog / change stream backend|
|5|Redis 族 AOF / replication stream backend|
|6|基于稳定 backend 构建数据恢复辅助能力|

## 开发

运行根模块测试：

```bash
go test ./...
```

运行 MySQL backend 测试：

```bash
go test ./mysql/...
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
