# Changelog

All notable changes to this project are tracked here.

## Unreleased

No user-visible changes yet.

## 0.1.0 - 2026-07-01

Initial public developer preview.

### Added

- Common `dblog` registry, event, filtering, and flashback APIs.
- MySQL, PostgreSQL, MongoDB, and Redis backend modules.
- Native package layout for backend registration, decoders, typed events, and
  plugin contracts.
- Parser tests, plugin tests, package documentation, lint configuration, and
  module-level CI coverage.

### Changed

- Repository module path and documentation now use `github.com/Infranite/go-dblog`.

### Fixed

- Backend modules can be tested independently with `GOWORK=off`.
- MySQL event registry duplicate registration now returns an error instead of
  panicking.
