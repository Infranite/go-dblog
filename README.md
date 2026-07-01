# go-dblog

[![CI](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml/badge.svg?branch=master)](https://github.com/Infranite/go-dblog/actions/workflows/dev-test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog)
[![Go Report Card](https://goreportcard.com/badge/github.com/Infranite/go-dblog)](https://goreportcard.com/report/github.com/Infranite/go-dblog)
[![License](https://img.shields.io/github/license/Infranite/go-dblog)](https://github.com/Infranite/go-dblog/blob/master/LICENSE)

`go-dblog` is a multi-module Go toolkit for parsing database change logs. It
provides a small common API for orchestration while each database-family backend
keeps its own native event model and dependency graph.

[中文说明](https://github.com/Infranite/go-dblog/blob/master/doc/doc_zh.md)

## Modules

Install only the backend you use.

| Module | Scope | Status |
|---|---|---|
| [`github.com/Infranite/go-dblog`](https://pkg.go.dev/github.com/Infranite/go-dblog) | Common API for multi-source orchestration | Supported |
| [`github.com/Infranite/go-dblog/mysql`](./mysql) | MySQL-family binlog parser: MySQL, MariaDB, MySQL-compatible dialects | Supported |
| [`github.com/Infranite/go-dblog/postgres`](./postgres) | PostgreSQL-family logical replication text parser | Supported |
| [`github.com/Infranite/go-dblog/mongo`](./mongo) | MongoDB-family oplog / change stream JSON parser | Supported |
| [`github.com/Infranite/go-dblog/redis`](./redis) | Redis-family AOF RESP parser | Supported |

Backend modules are split by database family instead of log format names. That
keeps imports predictable as each ecosystem grows its own dialects and
compatibility layers.

## Features

- One common event shape for MySQL, PostgreSQL, MongoDB, and Redis log streams.
- Explicit backend registration through `dblog.Registry`; no hidden imports or
  automatic global registration.
- Streaming decoders based on Go 1.23 iterators.
- Shared event helpers for source, position, filtering, and flashback output.
- Backend-native typed events for database-specific details.
- Plugin hooks inside native decoder packages for dialect-specific events and
  commands.
- Per-backend modules so callers install only the dependencies they need.

## Current Scope

The current public target is `v0.1.0` (unreleased): an offline parser release
for users who already have database log files, exported records, or captured
streams. Live replication readers are planned, but not part of this release
line.

| Backend | Supported input for `v0.1.0` | Not included yet |
|---|---|---|
| MySQL | Local MySQL-family binlog files | Online replication connection reader |
| PostgreSQL | Logical decoding text records | Logical replication protocol reader |
| MongoDB | Newline-delimited oplog or change stream JSON records | Live change stream reader |
| Redis | Redis AOF RESP array commands | Redis replication stream reader |

## Common API

The root module is intentionally small. It defines the shared event shape used
by orchestration code and leaves backend-specific parsing details inside each
backend module.

```bash
# These installs are intended for published v0.1.0 tags.
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

Use the common API for multi-source routing, shared filtering, CDC pipelines,
backend registration, and recovery tasks. Use backend-native APIs when you need
full database-specific event details. The current backend capability matrix and
CI evidence are tracked in [ROADMAP.md](./ROADMAP.md#capability-matrix).

## Examples

### Open a backend through the registry

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

### Generate flashback operations

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

## Backend Packages

Backend modules expose the same package shape:

| Package | Purpose |
|---|---|
| `<module>/backend` | Explicit registration with `dblog.Registry` |
| `<module>/decode/decoder` | Native streaming decoder and parser options |
| `<module>/decode/events/types` | Native event, change, command, and plugin contracts |

MySQL, PostgreSQL, MongoDB, and Redis keep backend-specific plugin hooks in
their native decoder packages. Use these hooks for dialect-specific commands,
event types, or compatibility behavior without changing the common API.

| Module | Documentation |
|---|---|
| `mysql` | [mysql/README.md](./mysql/README.md) |
| `postgres` | [postgres/README.md](./postgres/README.md) |
| `mongo` | [mongo/README.md](./mongo/README.md) |
| `redis` | [redis/README.md](./redis/README.md) |

## Roadmap

The detailed roadmap and capability matrix live in [ROADMAP.md](./ROADMAP.md).
Keep release planning there so version status, shipped capability, and CI
evidence have one source of truth.

Current public target: `v0.1.0` (unreleased), an offline parser developer
preview.

## Development and Contributing

Requirements:

- Go 1.23 or later.
- `golangci-lint` for local lint checks.
- Docker only when debugging fixture generation locally.

Run local unit tests:

```bash
make test
```

Run lint:

```bash
make lint
```

Run parser fuzz and benchmark smoke gates:

```bash
make fuzz-smoke
make bench-smoke
```

Full fixture-backed MySQL, MongoDB, PostgreSQL, and Redis integration tests run
in pull request CI. Pull requests merge through the protected `ci` and
`merge-policy` checks.

Run all fixture-backed integration tests locally when Docker is available:

```bash
make integration
```

When touching parser behavior, update tests in the affected backend module and
document user-visible behavior in the relevant README. Keep backend-specific
behavior inside that backend unless the common API genuinely needs it.

Fixture generation can be debugged locally when Docker is available:

```bash
./mysql/test/testdata/generate_mysql_binlog.sh mysql:8.4
./mongo/testdata/generate_mongo_oplog.sh mongo:7.0
./postgres/testdata/generate_postgres_logical.sh postgres:16
./redis/testdata/generate_redis_aof.sh redis:7.2
```

Contributions are handled through pull requests:

- Run `make test` and the affected module tests locally before opening a pull
  request.
- Keep parser behavior changes covered by tests in the affected backend.
- Update the relevant README when user-visible behavior changes.
- Full fixture-backed integration, fuzz smoke, benchmark smoke, lint, vet, and
  vulnerability checks run in CI.

## Versioning

Releases are managed with GitHub Releases and Git tags. The root module uses
`vX.Y.Z`; backend modules use module-prefixed tags such as `mysql/vX.Y.Z`,
`mongo/vX.Y.Z`, `postgres/vX.Y.Z`, and `redis/vX.Y.Z`. There is no standalone
release or changelog document in the repository: GitHub Releases are the public
release notes, and git history remains the detailed change log.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
