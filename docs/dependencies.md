# WOS Dependency Inventory

**Last reviewed:** 2026-10-05
**Active implementation wave:** Completion audit E00–E11, branch based on PR #13

This document records dependencies that are actually present in the repository. Planned dependencies from the architecture specification are not treated as installed or approved until the implementation wave that needs them.

## Go toolchain

- Module: `github.com/A1b3rt0M3rcad0/wos`
- Declared Go language/toolchain baseline: `go 1.27`
- CI verification observed on 2026-10-02: Go `1.27.1`

The `go.mod` directive is the source used by CI.

## Go module dependencies

### Direct

- `github.com/modelcontextprotocol/go-sdk v1.8.0`: API-only official MCP implementation; real stdio/Streamable HTTP client tests.
- `github.com/jackc/pgx/v5 v5.9.2`: PostgreSQL storage adapter; real database execution is required for acceptance.

- `github.com/ncruces/go-sqlite3 v0.35.6`
  - Scope: `packages/wos-core/storage/sqlite`
  - Purpose: CGO-free SQLite integration through `database/sql`, including the immediate transaction profile required by the SQLite writer-coordination contract.
  - Decision record: `docs/adr/0006-sqlite-driver-and-writer-acquisition.md`

### Indirect module graph

`go.mod` and `go.sum` contain the exact tidy graph. CI verifies both with a zero-diff `go mod tidy`. All external storage/transport dependencies remain outside Domain/Application/Ports. Apache 2.0 is the WOS license; the added Go SDK/driver retain their upstream notices/licenses.

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

## Browser verification

Playwright is a test-only dependency, pinned in `tests/web/package.json`. Browser binaries are not shipped with WOS; production UI uses no CDN or Node runtime. Browser test execution remains distinct from API session tests.

Optional OpenTelemetry has not been selected; metadata instrumentation uses standard `slog` and a Core port.

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
