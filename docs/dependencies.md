# WOS Dependency Inventory

**Last reviewed:** 2026-10-01  
**Active implementation wave:** Wave 01 — Public Core Foundation

This document records dependencies that are actually present in the repository. Planned dependencies from the architecture specification are not treated as installed or approved until the implementation wave that needs them.

## Go toolchain

- Module: `github.com/A1b3rt0M3rcad0/wos`
- Declared Go language/toolchain baseline: `go 1.27`
- CI verification observed on 2026-10-01: Go `1.27.1`

The `go.mod` directive is the source used by CI.

## Go module dependencies

Wave 01 has **no third-party Go module dependencies**.

The public Core and server bootstrap currently use the Go standard library only.

This is intentional. A dependency is added only when a concrete implementation requirement justifies it.

## CI dependencies

GitHub Actions currently uses:

- `actions/checkout@v7.0.1`
- `actions/setup-go@v7.0.0`

These are CI dependencies, not runtime dependencies of WOS.

## Planned but not yet selected

The architecture expects later implementation choices for areas such as:

- UUIDv7 generation implementation;
- SQLite driver;
- PostgreSQL driver;
- official Go MCP SDK;
- migration runner;
- optional OpenTelemetry integration.

No package/version for those areas is considered selected by this document. Their versions must be chosen, pinned and tested in the wave that introduces them.

## Dependency admission rules

Before adding a dependency:

1. identify the roadmap wave and concrete requirement it satisfies;
2. prefer the standard library when it provides a clear, maintainable implementation;
3. pin an explicit version in the repository;
4. verify license compatibility with the WOS license once the project license is selected;
5. keep the dependency out of `core/domain` unless it is genuinely part of a public domain contract and an ADR justifies it;
6. add tests covering the behavior for which the dependency was introduced;
7. update this file and `ROADMAP.md` in the same work.

Dependencies must not be added merely because they may be useful in a later wave.
