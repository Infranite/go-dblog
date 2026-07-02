# MySQL-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/mysql.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/mysql)

This module is the MySQL-family backend for `go-dblog`. It decodes MySQL binary
log files, reads MySQL replication streams, and keeps MySQL, MariaDB, and
MySQL-compatible dialect details in backend-native typed events.

[中文](./README.zh-CN.md)

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need MySQL-family binlog
parsing or replication stream reading.

## Installation

Current release:

```bash
go get github.com/Infranite/go-dblog/mysql@v0.2.0
```

The repository tag for this module is `mysql/v0.2.0`; callers use the semantic
version query above with `go get`.

Requirements:

- Go 1.25 or later.
- A MySQL-family binary log file, or a MySQL server with binary logging enabled
  and a user allowed to read replication streams.

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

## Documentation

| Topic | English | 中文 |
|---|---|---|
| Features, scope, package structure, event support, and plugins | [doc/FEATURES.md](./doc/FEATURES.md) | [doc/FEATURES.zh-CN.md](./doc/FEATURES.zh-CN.md) |
| Offline, live reader, filtering, plugin, and flashback examples | [doc/EXAMPLES.md](./doc/EXAMPLES.md) | [doc/EXAMPLES.zh-CN.md](./doc/EXAMPLES.zh-CN.md) |
| Project roadmap and release scope | [../doc/ROADMAP.md](../doc/ROADMAP.md#mysql-family) | [../doc/ROADMAP.zh-CN.md](../doc/ROADMAP.zh-CN.md#mysql-族) |
| Development and contribution flow | [../doc/DEVELOPMENT.md](../doc/DEVELOPMENT.md) | [../doc/DEVELOPMENT.zh-CN.md](../doc/DEVELOPMENT.zh-CN.md) |

## License

Apache License 2.0. See [LICENSE](../LICENSE).
