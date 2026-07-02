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
go get github.com/Infranite/go-dblog/mysql@v0.2.0
```

该 module 的仓库 tag 是 `mysql/v0.2.0`；调用方使用上面的 semantic version query。

要求：

- Go 1.25 或更新版本。
- MySQL-family binary log 文件，或启用了 binary logging 且允许读取 replication
  stream 的 MySQL server。

## Quick Start

```go
package main

import (
	"fmt"
	"strings"

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
		fmt.Printf("Got %s:\n\t", common.EventTypeName(event.Header.EventType))
		fmt.Println(event.Header)
		fmt.Println(strings.Repeat("=", 100))
	}
}
```

## 包结构

| Package | 用途 |
|---|---|
| `github.com/Infranite/go-dblog/mysql` | 常用 import 的 compatibility facade。 |
| `github.com/Infranite/go-dblog/mysql/backend` | 显式注册到 `dblog.Registry`。 |
| `github.com/Infranite/go-dblog/mysql/decode/decoder` | 原生 file decoder、live replication reader 和 parser options。 |
| `github.com/Infranite/go-dblog/mysql/decode/events/types` | 原生 binlog event 和 plugin types。 |

## 已支持

- MySQL 5.1+ binlog event parsing。
- 通过内置 dialect plugin 支持 MariaDB events。
- MySQL-compatible dialect events。
- 基于 `TABLE_MAP_EVENT` metadata 解码 row events。
- 在 auto/loose compatibility mode 中将 metadata-declared future events 保留为
  `*types.MetadataEvent`。
- Go iterator streaming 和 typed event filtering。
- 通过 `dblog.WithDSN` 打开 live replication stream。
- variable-width payload 的 copy-aware row value decoding。
- 通过根 registry 打开时支持 `dblog.WithCheckpoint`。
- 对完整 write、update、delete row image 输出 typed flashback row events。

## 暂不支持

- live reader 的 GTID auto-positioning。
- TLS-specific DSN 处理。
- 输入窗口缺少 `TABLE_MAP_EVENT` metadata 时重建完整 row value。
- incomplete row image、skipped columns 或 `PARTIAL_UPDATE_ROWS_EVENT` 的闪回。

## 支持输入

| 输入 | 状态 | CI 证据 |
|---|---|---|
| 来自 MySQL 5.6、5.7、8.0、8.4 的本地 MySQL-family binlog 文件 | 支持 | `mysql` CI matrix 从每个 image 生成真实 binlog fixture。 |
| MySQL、MariaDB 和 MySQL-compatible binlog event body | 支持 | Unit tests 覆盖 event decoders 和 MariaDB plugin。 |
| `FORMAT_DESCRIPTION_EVENT` metadata 声明的 unknown events | auto/loose compatibility mode 中作为 metadata events 保留 | Compatibility mode tests 和 fixture tests。 |
| malformed 或 undersized event headers | 拒绝 | `FuzzDecodeEventHeader` smoke target。 |
| 缺少前置 `TABLE_MAP_EVENT` 时解码 row events | 返回 event，并在 `DecodeError` 中记录原因，不重建 row value | `TestRowsEventWithoutPriorTableMapKeepsDecodeError`。 |
| 完整 `WRITE_ROWS_EVENT`、`UPDATE_ROWS_EVENT`、`DELETE_ROWS_EVENT` row image 的闪回 | 支持 typed reverse row events | Decoder tests 和 MySQL fixture CI。 |
| 在线 replication connection | 支持 | `TestLiveReplicationStream` 在 CI 中运行真实 `mysql:8.4` 容器。 |

## Live Replication Reader

注册 backend，并用 MySQL DSN 和 context 打开：

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

decoder, err := registry.Open(mysql.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("mysql://dblog:dblog@127.0.0.1:3306/?server_id=1002"),
)
```

DSN 支持可选 `binlog` 或 `file` 以及 `pos` query 参数。省略时从 server 当前 binary
log position 开始。GTID auto-positioning 和 TLS-specific DSN 处理不属于 `v0.2.0`
契约。取消 context 可停止 stream。Row details 需要 row-based binary logging。

## 闪回范围

`dblog.Flashbacks` 在原始 rows event 携带完整 row image 时输出 synthetic
`*events.Event`，body 为 typed `*types.BinRowsEvent`。

| 原始 event | 闪回 event |
|---|---|
| `WRITE_ROWS_EVENTv0/v1/v2` | 同版本 `DELETE_ROWS_EVENT` |
| `DELETE_ROWS_EVENTv0/v1/v2` | 同版本 `WRITE_ROWS_EVENT` |
| `UPDATE_ROWS_EVENTv0/v1/v2` | before/after rows 交换后的同版本 `UPDATE_ROWS_EVENT` |

缺少 table-map metadata、skipped columns 或 `PARTIAL_UPDATE_ROWS_EVENT` 不输出闪回。
如果输入窗口缺少前置 `TABLE_MAP_EVENT`，decoder 不猜测 column value，只在
`DecodeError` 中说明缺失 metadata。

## 插件

MariaDB plugin 默认启用。自定义 MySQL-family 方言可以注册 event plugin：

```go
fileDecoder, err := decoder.NewBinFileDecoder(
	"./mysql-bin.000001",
	decoder.WithEventPlugins(myPlugin),
)
```

完整 event support 表见 [English README](./README.md#event-support)。

## 开发

```bash
cd mysql && GOWORK=off go test ./...
make integration-mysql
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
