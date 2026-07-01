# go-dblog

<p align="center">
  <img src="./doc/assets/title-banner.svg" alt="go-dblog title banner">
</p>

[![CI](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml/badge.svg?branch=develop)](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/develop/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog)
[![Go Report Card](https://goreportcard.com/badge/github.com/Infranite/go-dblog)](https://goreportcard.com/report/github.com/Infranite/go-dblog)
[![License](https://img.shields.io/github/license/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/develop/LICENSE)

`go-dblog` is a multi-module Go toolkit for parsing database change logs. It
provides a small common API for orchestration while each database-family backend
keeps its own native event model and dependency graph.

[中文说明](https://github.com/Infranite/go-dblog/blob/develop/doc/doc_zh.md)

## Modules

Install only the backend you use.

| Module | Scope | Status |
|---|---|---|
| [`github.com/Infranite/go-dblog`](https://pkg.go.dev/github.com/Infranite/go-dblog) | Common API for multi-source orchestration | Supported |
| [`github.com/Infranite/go-dblog/mysql`](./mysql) | MySQL-family binlog parser: MySQL, MariaDB, MySQL-compatible dialects | Supported |
| `github.com/Infranite/go-dblog/postgres` | PostgreSQL-family logical replication / WAL parser | Planned |
| `github.com/Infranite/go-dblog/mongo` | MongoDB-family oplog / change stream parser | Planned |
| `github.com/Infranite/go-dblog/redis` | Redis-family AOF / replication stream parser | Planned |

Backend modules are split by database family instead of log format names. That
keeps imports predictable as each ecosystem grows its own dialects and
compatibility layers.

## Common API

The root module is intentionally small. It defines the shared event shape used
by orchestration code and leaves backend-specific parsing details inside each
backend module.

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

Use the common API for multi-source routing, shared filtering, CDC pipelines,
and recovery workflows. Use backend-native APIs when you need full
database-specific event details.

## Backends

The first backend is the MySQL-family module:

```bash
go get github.com/Infranite/go-dblog/mysql
```

See [mysql/README.md](./mysql/README.md) for installation, examples,
compatibility modes, dialect plugins, and supported event tables.

## Roadmap

| Phase | Scope |
|---|---|
| 1 | MySQL-family replication connection reader |
| 2 | PostgreSQL-family logical replication backend |
| 3 | Shared CDC helpers after at least two real backends exist |
| 4 | MongoDB-family oplog / change stream backend |
| 5 | Redis-family AOF / replication stream backend |
| 6 | Recovery helpers built on stable backend parsers |

## Development

Run root module tests:

```bash
go test ./...
```

Run MySQL backend tests:

```bash
go test ./mysql/...
```

## Contributing

Issues and pull requests are welcome. Keep backend changes focused, preserve
native event details, and add parser tests for new log formats.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
