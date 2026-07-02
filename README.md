# go-dblog

[![CI](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml/badge.svg?branch=master)](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog)
[![Go Report Card](https://goreportcard.com/badge/github.com/Infranite/go-dblog)](https://goreportcard.com/report/github.com/Infranite/go-dblog)
[![License](https://img.shields.io/github/license/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/LICENSE)

`go-dblog` is a multi-module Go toolkit for parsing database change logs. The
root module defines the shared event, registry, checkpoint, filtering, and
flashback contracts; each product backend keeps its native event model and
dependencies in its own module.

[中文](./doc/README.zh-CN.md)

## Product Index

Install only the backend you use.

| Product | Module | Details |
|---|---|---|
| Common API | `github.com/Infranite/go-dblog` | This README |
| MySQL family | `github.com/Infranite/go-dblog/mysql` | [English](./mysql/README.md) / [中文](./mysql/README.zh-CN.md) |
| PostgreSQL family | `github.com/Infranite/go-dblog/postgres` | [English](./postgres/README.md) / [中文](./postgres/README.zh-CN.md) |
| MongoDB family | `github.com/Infranite/go-dblog/mongo` | [English](./mongo/README.md) / [中文](./mongo/README.zh-CN.md) |
| Redis family | `github.com/Infranite/go-dblog/redis` | [English](./redis/README.md) / [中文](./redis/README.zh-CN.md) |

Current supported and unsupported source details live in
[doc/ROADMAP.md](./doc/ROADMAP.md).

## Features

- Backend-neutral `dblog.Event` shape for routing mixed database log streams.
- Explicit backend registration through `dblog.Registry`.
- Streaming decoders built on Go iterator APIs.
- Shared helpers for source metadata, positions, checkpoint resume, filtering,
  and safe flashback output.
- Backend-native typed events for database-specific fields.
- Plugin hooks inside backend decoder packages for dialect-specific records.
- Separate Go modules so callers do not install unused database dependencies.

## Install

No public tags have been published yet. Until the first `v0.2.0` tag set
exists, evaluate the project from a checked-out branch or commit.

```bash
# After the v0.2.0 tag set is published:
go get github.com/Infranite/go-dblog@v0.2.0
go get github.com/Infranite/go-dblog/mysql@v0.2.0
go get github.com/Infranite/go-dblog/postgres@v0.2.0
go get github.com/Infranite/go-dblog/mongo@v0.2.0
go get github.com/Infranite/go-dblog/redis@v0.2.0
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

## Documentation

| Topic | English | 中文 |
|---|---|---|
| Project overview | This README | [doc/README.zh-CN.md](./doc/README.zh-CN.md) |
| Roadmap and product scope | [doc/ROADMAP.md](./doc/ROADMAP.md) | [doc/ROADMAP.zh-CN.md](./doc/ROADMAP.zh-CN.md) |
| Development and contribution flow | [doc/DEVELOPMENT.md](./doc/DEVELOPMENT.md) | [doc/DEVELOPMENT.zh-CN.md](./doc/DEVELOPMENT.zh-CN.md) |
| Security policy | [doc/SECURITY.md](./doc/SECURITY.md) | [doc/SECURITY.zh-CN.md](./doc/SECURITY.zh-CN.md) |

GitHub Releases and git tags are the public release record. Git history is the
detailed change log; this repository does not maintain separate release notes
or changelog files.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
