# go-dblog

[![CI](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml/badge.svg?branch=master)](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog)
[![Go Report Card](https://goreportcard.com/badge/github.com/Infranite/go-dblog)](https://goreportcard.com/report/github.com/Infranite/go-dblog)
[![License](https://img.shields.io/github/license/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/LICENSE)

`go-dblog` is a multi-module Go toolkit for parsing database change logs. The
root module defines shared event, registry, checkpoint, filtering, and safe
flashback/recovery contracts; each backend keeps product-specific parsing, live
reading, typed events, and extension hooks in its own module.

[中文](./doc/README.zh-CN.md)

## Product Index

Install only the backend you use.

| Product | Module | README | Features | Examples |
|---|---|---|---|---|
| Common API | `github.com/Infranite/go-dblog` | This README | [Roadmap scope](./doc/ROADMAP.md#common-api) | Minimal example below |
| MySQL family | `github.com/Infranite/go-dblog/mysql` | [English](./mysql/README.md) / [中文](./mysql/README.zh-CN.md) | [English](./mysql/doc/FEATURES.md) / [中文](./mysql/doc/FEATURES.zh-CN.md) | [English](./mysql/doc/EXAMPLES.md) / [中文](./mysql/doc/EXAMPLES.zh-CN.md) |
| PostgreSQL family | `github.com/Infranite/go-dblog/postgres` | [English](./postgres/README.md) / [中文](./postgres/README.zh-CN.md) | [English](./postgres/doc/FEATURES.md) / [中文](./postgres/doc/FEATURES.zh-CN.md) | [English](./postgres/doc/EXAMPLES.md) / [中文](./postgres/doc/EXAMPLES.zh-CN.md) |
| MongoDB family | `github.com/Infranite/go-dblog/mongo` | [English](./mongo/README.md) / [中文](./mongo/README.zh-CN.md) | [English](./mongo/doc/FEATURES.md) / [中文](./mongo/doc/FEATURES.zh-CN.md) | [English](./mongo/doc/EXAMPLES.md) / [中文](./mongo/doc/EXAMPLES.zh-CN.md) |
| Redis family | `github.com/Infranite/go-dblog/redis` | [English](./redis/README.md) / [中文](./redis/README.zh-CN.md) | [English](./redis/doc/FEATURES.md) / [中文](./redis/doc/FEATURES.zh-CN.md) | [English](./redis/doc/EXAMPLES.md) / [中文](./redis/doc/EXAMPLES.zh-CN.md) |

## Core Features

- Backend-neutral `dblog.Event` shape for mixed database log streams.
- Explicit backend registration through `dblog.Registry`.
- Streaming decoders built on Go iterator APIs.
- Shared source metadata, position, checkpoint resume, filtering, and safe
  flashback/recovery helpers.
- Backend-native typed events for database-specific fields.
- Plugin hooks inside backend decoder packages for compatible dialects and
  product-specific records.
- Separate Go modules so callers do not install unused database dependencies.

## Install

The current public tag set is `v0.4.0`.

```bash
go get github.com/Infranite/go-dblog@v0.4.0
go get github.com/Infranite/go-dblog/mysql@v0.4.0
go get github.com/Infranite/go-dblog/postgres@v0.4.0
go get github.com/Infranite/go-dblog/mongo@v0.4.0
go get github.com/Infranite/go-dblog/redis@v0.4.0
```

## Minimal Example

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

Use the common API for multi-source routing, shared filtering, CDC pipelines,
backend registration, and recovery tasks. Use backend-native APIs when you need
database-specific event fields.

## Project Docs

| Topic | English | 中文 |
|---|---|---|
| Project overview | This README | [doc/README.zh-CN.md](./doc/README.zh-CN.md) |
| Recovery cookbook | [doc/RECOVERY.md](./doc/RECOVERY.md) | [doc/RECOVERY.zh-CN.md](./doc/RECOVERY.zh-CN.md) |
| CI evidence | [doc/CI.md](./doc/CI.md) | [doc/CI.zh-CN.md](./doc/CI.zh-CN.md) |
| Roadmap and product scope | [doc/ROADMAP.md](./doc/ROADMAP.md) | [doc/ROADMAP.zh-CN.md](./doc/ROADMAP.zh-CN.md) |
| Development and contribution flow | [doc/DEVELOPMENT.md](./doc/DEVELOPMENT.md) | [doc/DEVELOPMENT.zh-CN.md](./doc/DEVELOPMENT.zh-CN.md) |
| Security policy | [doc/SECURITY.md](./doc/SECURITY.md) | [doc/SECURITY.zh-CN.md](./doc/SECURITY.zh-CN.md) |

GitHub Releases and git tags are the public release record. Git history is the
detailed change log; this repository does not maintain separate release notes
or changelog files.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
