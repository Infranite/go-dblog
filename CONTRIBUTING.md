# Contributing

`go-dblog` is a multi-module Go repository. Keep changes scoped to the backend
or common API they affect, and add tests at the same package level as the
behavior being changed.

## Prerequisites

- Go 1.23 or newer.
- `golangci-lint` for local lint checks.
- Docker only when debugging fixture generation locally.

## Local Checks

Local checks are intentionally unit-only and do not require database services:

```bash
make test
make lint
```

`make test` runs the root module plus MySQL, MongoDB, PostgreSQL, and Redis as
independent modules with `GOWORK=off` and `-short`. Fixture-backed integration
tests are skipped in this path.

To debug a fixture locally, run the relevant generator explicitly:

```bash
./mysql/test/testdata/generate_mysql_binlog.sh mysql:8.4
./mongo/testdata/generate_mongo_oplog.sh mongo:7.0
./postgres/testdata/generate_postgres_logical.sh postgres:16
./redis/testdata/generate_redis_aof.sh redis:7.2
```

## Pull Requests

- Document user-visible behavior in the relevant README.
- Add or update parser tests for every supported log shape.
- Add benchmark coverage when changing parser hot paths.
- Keep backend-specific behavior inside that backend module unless the common
  API genuinely needs it.
- Open a pull request into `master`; do not push directly to `master`.
- Enable auto-merge after the pull request is ready. The required `ci` check
  aggregates lint, vet, vulnerability scanning, unit tests, and real
  fixture-backed MySQL, MongoDB, PostgreSQL, and Redis integration tests.
