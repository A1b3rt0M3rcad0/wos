# WOS Dependency Inventory

**Last reviewed:** 2026-10-02  
**Active implementation wave:** Wave 04 — SQLite Persistence and Migrations (complete in PR #4; not yet merged)

This document records dependencies that are actually present in the repository. Planned dependencies from the architecture specification are not treated as installed or approved until the implementation wave that needs them.

## Go toolchain

- Module: `github.com/A1b3rt0M3rcad0/wos`
- Declared Go language/toolchain baseline: `go 1.27`
- CI verification observed on 2026-10-02: Go `1.27.1`

The `go.mod` directive is the source used by CI.

## Go module dependencies

### Direct

- `github.com/ncruces/go-sqlite3 v0.35.6`
  - Scope: `packages/wos-core/storage/sqlite`
  - Purpose: CGO-free SQLite integration through `database/sql`, including the immediate transaction profile required by the SQLite writer-coordination contract.
  - Decision record: `docs/adr/0006-sqlite-driver-and-writer-acquisition.md`

### Indirect module graph

The current tidy `go.mod` includes:

- `github.com/ncruces/go-sqlite3-wasm/v6 v6.3.35304`
- `github.com/ncruces/julianday v1.0.0`
- `golang.org/x/sys v0.48.0`

`go.sum` is committed and CI runs `go mod tidy` followed by a zero-diff check.

The public Domain/Application/Ports layers do not import the SQLite dependency. It remains confined to the storage adapter.

## CI dependencies

GitHub Actions currently uses:

- `actions/checkout@v7.0.1`
- `actions/setup-go@v7.0.0`

These are CI dependencies, not runtime dependencies of WOS.

## Implemented without a third-party package

Wave 04 uses repository-owned code for:

- numbered embedded SQL migrations;
- migration checksum validation;
- UTC microsecond timestamp codec;
- SQLite backup/restore orchestration.

No external migration framework was added.

## Planned but not yet selected

The architecture still expects later implementation choices for areas such as:

- PostgreSQL driver;
- official Go MCP SDK/profile;
- optional OpenTelemetry integration.

No package/version for those areas is considered selected by this document. Their versions must be chosen, pinned and tested in the wave that introduces them.

## Dependency admission rules

Before adding a dependency:

1. identify the roadmap wave and concrete requirement it satisfies;
2. prefer the standard library when it provides a clear, maintainable implementation;
3. pin an explicit version in the repository;
4. verify license compatibility with the WOS license once the project license is selected;
5. keep the dependency out of `packages/wos-core/domain` unless it is genuinely part of a public domain contract and an ADR justifies it;
6. add tests covering the behavior for which the dependency was introduced;
7. update this file and `ROADMAP.md` in the same work.

Dependencies must not be added merely because they may be useful in a later wave.
