# go-dblog

[![CI](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml/badge.svg?branch=master)](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog)
[![Go Report Card](https://goreportcard.com/badge/github.com/Infranite/go-dblog)](https://goreportcard.com/report/github.com/Infranite/go-dblog)
[![License](https://img.shields.io/github/license/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/LICENSE)

`go-dblog` 是一个面向数据库变更日志解析、CDC 和数据恢复工作流的多模块 Go
工具包。根模块只提供公共事件模型和跨 backend 编排 API；具体数据库族保留自己的
原生事件结构、解析细节和依赖图。

[English](https://github.com/Infranite/go-dblog/blob/master/README.md)

## 模块

只安装实际使用的 backend。

|模块|范围|状态|
|---|---|---|
|[`github.com/Infranite/go-dblog`](https://pkg.go.dev/github.com/Infranite/go-dblog)|多数据源编排公共 API|已支持|
|[`github.com/Infranite/go-dblog/mysql`](../mysql)|MySQL 族 binlog：MySQL、MariaDB、MySQL-compatible 方言|已支持|
|[`github.com/Infranite/go-dblog/postgres`](../postgres)|PostgreSQL 族 logical replication 文本解析|已支持|
|[`github.com/Infranite/go-dblog/mongo`](../mongo)|MongoDB 族 oplog / change stream JSON 解析|已支持|
|[`github.com/Infranite/go-dblog/redis`](../redis)|Redis 族 AOF RESP 解析|已支持|

backend 按数据库族命名，而不是按某一种日志格式命名。MySQL 族使用 `mysql` 模块；
PostgreSQL 族后续使用 `postgres` 模块，以便容纳兼容方言和生态扩展。

## 功能

- MySQL、PostgreSQL、MongoDB、Redis 使用统一公共事件形态。
- 通过 `dblog.Registry` 显式注册 backend；不依赖隐藏 import 或自动全局注册。
- 基于 Go 1.23 iterator 的流式 decoder。
- 公共层提供 source、position、过滤和闪回辅助能力。
- 各 backend 保留数据库原生 typed event，承载数据库特有细节。
- 原生 decoder 包提供插件入口，用于方言事件和命令扩展。
- 按 backend 拆分 module，调用方只安装实际需要的依赖。

## 当前范围

`v0.1.0` 是离线解析版本。它适合已经有数据库日志文件、导出记录或捕获流的用户。
在线 replication reader 已规划，但不属于第一个公开版本。

|Backend|`v0.1.0` 支持输入|暂不包含|
|---|---|---|
|MySQL|本地 MySQL 族 binlog 文件|在线 replication 连接 reader|
|PostgreSQL|logical decoding 文本记录|logical replication 协议 reader|
|MongoDB|按行分隔的 oplog 或 change stream JSON 记录|实时 change stream reader|
|Redis|Redis AOF RESP 数组命令|Redis replication stream reader|

## 公共层 API

```bash
go get github.com/Infranite/go-dblog
go get github.com/Infranite/go-dblog/mysql
go get github.com/Infranite/go-dblog/postgres
go get github.com/Infranite/go-dblog/mongo
go get github.com/Infranite/go-dblog/redis
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
backend 注册、CDC 或恢复流水线时，再接入根模块公共 API。

## 示例

### 通过 registry 打开 backend

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

	for event, err := range dblog.Filter(
		decoder.Events(),
		dblog.ByDriver(redis.Driver),
		dblog.ByKind(redis.CommandSAdd),
	) {
		if err != nil {
			panic(err)
		}
		command := event.Body().(redis.Command)
		fmt.Println(event.Kind(), command.Args)
	}
}
```

### 生成闪回操作

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
		dblog.WithReader(strings.NewReader("*4\r\n$4\r\nHSET\r\n$6\r\nuser:1\r\n$4\r\nname\r\n$3\r\nAda\r\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			panic(err)
		}
		fmt.Println(op)
	}
}
```

## Backend 包结构

```bash
go get github.com/Infranite/go-dblog/mysql
go get github.com/Infranite/go-dblog/postgres
go get github.com/Infranite/go-dblog/mongo
go get github.com/Infranite/go-dblog/redis
```

各 backend module 暴露一致的包结构：

|包|用途|
|---|---|
|`<module>/backend`|显式注册到 `dblog.Registry`|
|`<module>/decode/decoder`|原生流式 decoder 和解析选项|
|`<module>/decode/events/types`|原生事件、变更、命令和插件契约|

MySQL、PostgreSQL、MongoDB 和 Redis 都在各自原生 decoder 包中保留插件入口。
方言命令、特殊事件类型和兼容行为通过插件扩展，不污染公共层 API。

|模块|文档|
|---|---|
|`mysql`|[`mysql/README.md`](../mysql/README.md)|
|`postgres`|[`postgres/README.md`](../postgres/README.md)|
|`mongo`|[`mongo/README.md`](../mongo/README.md)|
|`redis`|[`redis/README.md`](../redis/README.md)|

## 路线图

完整路线图见 [`ROADMAP.md`](../ROADMAP.md)。当前发布线：

|版本|状态|主题|
|---|---|---|
|`v0.1.0`|进行中|离线 parser developer preview|
|`v0.2.0`|计划中|兼容性加固|
|`v0.3.0`|计划中|在线 reader|
|`v0.4.0`|计划中|恢复工作流|
|`v0.5.0`|计划中|工程成熟度|
|`v1.0.0`|候选|稳定公共 API|

## 开发

运行本地单测：

```bash
make test
```

运行 lint：

```bash
make lint
```

完整的 MySQL、MongoDB、PostgreSQL、Redis 真实 fixture 集成测试在 pull request CI
中运行，合入受受保护的 `ci` 检查约束。有 Docker 时可本地调试 MySQL fixture：

```bash
make integration-mysql
```

## 发布

第一个公开版本目标是 `v0.1.0`。backend module 依赖同版本根模块，必须先打根模块 tag，
再打 backend module tag。详见 [`RELEASE.md`](../RELEASE.md)。

## License

Apache License 2.0. See [LICENSE](../LICENSE).
