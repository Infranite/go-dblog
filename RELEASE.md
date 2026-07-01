# Release

The first public version is `v0.1.0`.

## Checks

Run these before tagging:

```bash
make lint
make test
```

Tag only from a pull request merge commit that passed the protected `ci` check.
The `ci` check includes lint, vet, vulnerability scanning, unit tests, and real
fixture-backed MySQL, MongoDB, PostgreSQL, and Redis integration tests.

## Tag Order

Backend modules require the root module at the same version, so tag the root
module first.

The backend `go.mod` files keep a local `replace github.com/Infranite/go-dblog
=> ..` so repository checks work before tags exist. Consumers ignore dependency
module `replace` directives and resolve the required root tag.

```bash
git tag v0.1.0
git tag mysql/v0.1.0
git tag mongo/v0.1.0
git tag postgres/v0.1.0
git tag redis/v0.1.0
git push origin v0.1.0 mysql/v0.1.0 mongo/v0.1.0 postgres/v0.1.0 redis/v0.1.0
```

Use module-prefixed tags for backend modules because each backend has its own
`go.mod`.
