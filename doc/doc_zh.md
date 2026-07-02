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
|[`github.com/Infranite/go-dblog/postgres`](../postgres)|PostgreSQL 族 logical decoding 解析和 SQL slot reader|已支持|
|[`github.com/Infranite/go-dblog/mongo`](../mongo)|MongoDB 族 oplog / change stream JSON 解析|已支持|
|[`github.com/Infranite/go-dblog/redis`](../redis)|Redis 族 AOF RESP 解析|已支持|

backend 按数据库族命名，而不是按某一种日志格式命名。MySQL 族使用 `mysql` 模块；
PostgreSQL 族后续使用 `postgres` 模块，以便容纳兼容方言和生态扩展。

## 功能

- MySQL、PostgreSQL、MongoDB、Redis 使用统一公共事件形态。
- 通过 `dblog.Registry` 显式注册 backend；不依赖隐藏 import 或自动全局注册。
- 基于 Go 1.23 iterator 的流式 decoder。
- 公共层提供 source、position、checkpoint resume、过滤和闪回辅助能力。
- 各 backend 保留数据库原生 typed event，承载数据库特有细节。
- 原生 decoder 包提供插件入口，用于方言事件和命令扩展。
- 按 backend 拆分 module，调用方只安装实际需要的依赖。

## 当前范围

当前公开目标是 `v0.1.0`：面向已有数据库日志文件、导出记录、捕获流或 PostgreSQL
`test_decoding` logical slot 用户的解析版本。首个 tag 发布前，建议使用检出的分支
进行评估。大部分在线 replication reader 仍在后续发布线中规划。

|Backend|`v0.1.0` 支持输入|暂不包含|
|---|---|---|
|MySQL|本地 MySQL 族 binlog 文件|在线 replication 连接 reader|
|PostgreSQL|logical decoding 文本记录；基于 `test_decoding` 的 SQL logical slot 轮询|wire-level logical replication 协议 reader|
|MongoDB|按行分隔的 oplog 或 change stream JSON 记录|实时 change stream reader|
|Redis|Redis AOF RESP 数组命令|Redis replication stream reader|

## 公共层 API

```bash
# 这些安装命令面向公开 tags 发布后使用。
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
backend 注册、CDC 或恢复流水线时，再接入根模块公共 API。当前 backend 能力矩阵和
CI 证据记录在 [`ROADMAP.md`](../ROADMAP.md#capability-matrix)。

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
		dblog.WithReader(strings.NewReader("*2\r\n$4\r\nINCR\r\n$7\r\ncounter\r\n")),
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

### 从 checkpoint 恢复

```go
checkpoint := dblog.CheckpointOf(lastProcessedEvent)

decoder, err := registry.Open(redis.Driver,
	dblog.WithReader(strings.NewReader(aof)),
	dblog.WithCheckpoint(checkpoint),
)
```

## Backend 包结构

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

详细路线图和能力矩阵见 [`ROADMAP.md`](../ROADMAP.md)。版本状态、已发布能力和 CI
证据只在 roadmap 中维护，避免多处表格漂移。

当前公开目标：`v0.1.0`，parser developer preview 的首个 tag 集合。

## 开发

要求：

- Go 1.23 或更新版本。
- 本地 lint 需要 `golangci-lint`。
- Docker 只在本地调试 fixture 生成时需要。

运行本地单测：

```bash
make test
```

运行 lint：

```bash
make lint
```

运行 parser fuzz 和 benchmark smoke 门禁：

```bash
make fuzz-smoke
make bench-smoke
```

完整的 MySQL、MongoDB、PostgreSQL、Redis 真实 fixture 集成测试在 pull request CI
中运行，合入受 `ci` 和 `merge-policy` 检查约束。

有 Docker 时可本地运行完整 fixture 集成测试：

```bash
make integration
```

修改 parser 行为时，在对应 backend module 更新测试，并在相关 README 中记录用户可见
行为。backend 特有逻辑应留在对应 backend，除非公共 API 确实需要承载。

有 Docker 时可本地调试 fixture：

```bash
./mysql/test/testdata/generate_mysql_binlog.sh mysql:8.4
./mongo/testdata/generate_mongo_oplog.sh mongo:7.0
./postgres/testdata/generate_postgres_logical.sh postgres:16
./postgres/testdata/run_postgres_live.sh postgres:16
./redis/testdata/generate_redis_aof.sh redis:7.2
```

贡献通过 pull request 处理：

- 提交前本地运行 `make test` 和受影响 module 的测试。
- parser 行为变更必须在对应 backend 中补测试。
- 用户可见行为变化需要同步更新相关 README。
- 完整 fixture 集成测试、fuzz smoke、benchmark smoke、lint、vet 和漏洞扫描由 CI 运行。

## License

Apache License 2.0. See [LICENSE](../LICENSE).
