# Development

This document covers local checks, fixture generation, pull requests, and
versioning for `go-dblog`.

[中文](./DEVELOPMENT.zh-CN.md)

## Requirements

- Go 1.25 or later.
- `golangci-lint` for local lint checks.
- Docker only when debugging fixture generation or integration tests locally.

## Local Checks

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
in pull request CI. Run them locally only when Docker is available:

```bash
make integration
```

CI also publishes a tested backend/version matrix and parser benchmark history
as workflow artifacts. See [CI.md](./CI.md) for the current matrix and artifact
names.

## Fixture Debugging

Use these commands when you need to debug CI-style fixture generation locally:

```bash
./mysql/test/testdata/generate_mysql_binlog.sh mysql:8.4
./mysql/test/testdata/run_mysql_live.sh mysql:8.4
./mongo/testdata/generate_mongo_oplog.sh mongo:7.0
./mongo/testdata/run_mongo_live.sh mongo:7.0
./postgres/testdata/generate_postgres_logical.sh postgres:16
./postgres/testdata/run_postgres_live.sh postgres:16
./redis/testdata/generate_redis_aof.sh redis:7.2
./redis/testdata/run_redis_live.sh redis:7.2
```

## Pull Requests

Pull requests are the contribution path. A standalone `CONTRIBUTING.md` is not
maintained.

- Run `make test` and the affected module tests locally before opening a pull
  request.
- Keep parser behavior changes covered by tests in the affected backend.
- Update the relevant README or `doc/` page when user-visible behavior changes.
- Full fixture-backed integration, fuzz smoke, benchmark smoke, lint, vet, and
  vulnerability checks run in CI.
- CI publishes tested matrix and benchmark history artifacts for release
  evidence.
- Pull requests merge through the protected `ci` and `merge-policy` checks.

## Versioning

- Root module tags use `vX.Y.Z`.
- Backend module tags use `mysql/vX.Y.Z`, `postgres/vX.Y.Z`,
  `mongo/vX.Y.Z`, and `redis/vX.Y.Z`.
- Backend modules track the root module version for every release tag.
- GitHub Releases and git tags are the public release record.
- Git history is the detailed change log; this repository does not maintain
  separate release notes or changelog files.
