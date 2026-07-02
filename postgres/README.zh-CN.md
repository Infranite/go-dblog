# PostgreSQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/postgres.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/postgres)

该 module 是 `go-dblog` 的 PostgreSQL 族 backend。它解析 logical decoding 文本记录，
通过 SQL slot polling 或 PostgreSQL replication protocol 读取 live `test_decoding`
输出，并用 typed events 暴露 transaction 和 row changes。

[English](./README.md)

需要多数据源编排时使用根 [`go-dblog`](../README.md) module。只需要 PostgreSQL 族
logical decoding 文本解析时可直接使用本 module。

## 安装

当前还没有发布公开 tag。首个 `v0.2.0` tag 集合发布后：

```bash
go get github.com/Infranite/go-dblog/postgres@v0.2.0
```

该 module 的仓库 tag 是 `postgres/v0.2.0`；调用方使用上面的 semantic version query。

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

## 包结构

| Package | 用途 |
|---|---|
| `github.com/Infranite/go-dblog/postgres` | 常用 import 的 compatibility facade。 |
| `github.com/Infranite/go-dblog/postgres/backend` | 显式注册到 `dblog.Registry`。 |
| `github.com/Infranite/go-dblog/postgres/decode/decoder` | 原生 streaming decoder、line parser 和 plugin options。 |
| `github.com/Infranite/go-dblog/postgres/decode/events/types` | 原生 transaction、change、event 和 plugin types。 |

## 已支持

- `BEGIN` 和 `COMMIT` transaction records。
- PostgreSQL logical decoding text form 的 row changes。
- `null`、booleans、integers、floats、quoted strings 的 scalar parsing。
- 有界 scanner buffer 的 streaming line decoder。
- PostgreSQL `test_decoding` 输出的 live SQL logical slot reader。
- PostgreSQL `test_decoding` 输出的 wire-level logical replication reader。
- 通过 `postgres/backend` 集成根 registry。
- 通过根 registry 打开时支持 `dblog.WithCheckpoint`。
- 对 insert、delete 和具备完整 old/new tuple 数据的 update 生成 SQL 闪回。
- 面向 PostgreSQL-compatible source 额外 line types 的 event plugin。

backend driver name 是 `pg`，module path 保持 `postgres`。

## 暂不支持

- `pgoutput` binary relation/tuple messages。
- raw WAL/page 解码。
- `test_decoding` 以外的 output plugin，除非自定义 text event plugin 处理。
- old tuple 未覆盖所有 new tuple column 时的 update 闪回。

## 支持输入

| 输入 | 状态 | CI 证据 |
|---|---|---|
| logical decoding text output 中的 `BEGIN` 和 `COMMIT` records | 支持 | Unit tests 和 `postgres:16` fixture job。 |
| `table schema.table: OPERATION: col[type]:value` 形式的 row changes | 支持 | Unit tests、fixture job 和 `FuzzParseLine` smoke target。 |
| `UPDATE: old-key: ... new-tuple: ...` 且具备完整 old tuple data | 支持 | Unit tests、fuzz seed 和启用 `REPLICA IDENTITY FULL` 的 fixture job。 |
| `test_decoding` live SQL logical slot polling | 支持 | `TestLiveLogicalDecoding` 在真实 `postgres:16` 容器中运行。 |
| `test_decoding` wire-level logical replication | 支持 | `TestWireLogicalReplication` 在真实 `postgres:16` 容器中运行。 |
| empty table 或 operation names | 拒绝 | Parser tests 和 fuzz smoke target。 |
| `pgoutput` binary relation/tuple messages | 不支持，并由 text parser 拒绝 | `TestParseLineRejectsPgoutputBinaryMessages`。 |

## Live Readers

SQL slot reader：

```go
decoder, err := registry.Open(postgres.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"),
	dblog.WithSource(dblog.Source{Name: "dblog_slot"}),
)
```

Wire-level replication reader：

```go
decoder, err := registry.Open(postgres.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable&replication=database"),
	dblog.WithSource(dblog.Source{Name: "dblog_slot"}),
)
```

两种 live reader 都解析同一个 `test_decoding` 文本格式。`v0.2.0` 的 live reader 是
text-oriented：请使用 `test_decoding`，或通过 `decoder.WithEventPlugins` 归一化自定义
文本 output plugin；binary `pgoutput` relation 和 tuple messages 不会被解码。取消
context 可停止读取。

## 闪回范围

| Event | 闪回输出 |
|---|---|
| `insert` | `DELETE FROM ... WHERE ...;` |
| `update` 且包含完整 `old-key` 和 `new-tuple` columns | `UPDATE ... SET old_values WHERE new_values;` |
| `delete` | `INSERT INTO ... VALUES ...;` |
| 缺少完整 old/new tuple data 的 `update`、`begin`、`commit` | 不输出闪回。 |

## 插件

使用 `decoder.WithEventPlugins` 处理内置 logical decoding 文本记录以外的 line family。
内置 parser 不接受某行后，plugin 会收到原始 line；plugin 应输出 backend-native
`types.Event`，这样 root adapter 能保留 source、position 和 checkpoint 行为。示例见
[English README](./README.md#event-plugins)。

## 开发

```bash
cd postgres && GOWORK=off go test ./...
make integration-postgres
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
