# Contributing

`go-dblog` is a multi-module Go repository. Keep changes scoped to the backend
or common API they affect, and add tests at the same package level as the
behavior being changed.

## Prerequisites

- Go 1.23 or newer.
- `golangci-lint` for local lint checks.
- Docker only when regenerating MySQL binlog fixtures.

## Local Checks

Run the same checks used by CI:

```bash
make lint
make test
make test-mysql
```

`make test` runs the root module plus MongoDB, PostgreSQL, and Redis as
independent modules with `GOWORK=off`. `make test-mysql` expects the MySQL
fixture under `mysql/test/testdata/mysql-bin.000004`; without it, fixture-backed
tests are skipped. CI generates the fixture for the MySQL matrix before running
the MySQL module tests.

To regenerate the MySQL fixture:

```bash
./mysql/test/testdata/generate_mysql_binlog.sh mysql:8.4
```

## Pull Requests

- Document user-visible behavior in the relevant README.
- Add or update parser tests for every supported log shape.
- Add benchmark coverage when changing parser hot paths.
- Keep backend-specific behavior inside that backend module unless the common
  API genuinely needs it.
