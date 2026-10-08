# WOS Dependency Inventory

**Last reviewed:** 2026-10-07
**Active implementation wave:** Waves 01–18 validation candidate

This document records dependencies that are actually present in the repository. Planned dependencies from the architecture specification are not treated as installed or approved until the implementation wave that needs them.

## Go toolchain

- Module: `github.com/A1b3rt0M3rcad0/wos`
- Declared Go language/toolchain baseline: `go 1.27`
- Local and CI verification baseline: Go `1.27.1`

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
- `actions/setup-node@v7.0.0` and `actions/upload-artifact@v7.0.2` for package/release verification; Node 24.19.0 and npm 11.9.0 pin artifact generation.

Release/version/installer code uses Node standard libraries only. npm/OIDC, GitHub API/gh and Docker/GHCR are distribution services/tools, not Core or agent-runtime dependencies. Local workflow lint uses actionlint 1.7.7 as verification tooling; it is not shipped.

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

The portable skills installer uses Node >=22 standard libraries only, with no runtime dependencies and no npm lifecycle scripts. Distribution tooling uses existing Go and Node tools. Node is needed for npm wrappers, not for the standalone Go service, container or Core library.

Before adding a dependency:

1. identify the roadmap wave and concrete requirement it satisfies;
2. prefer the standard library when it provides a clear, maintainable implementation;
3. pin an explicit version in the repository;
4. verify license compatibility with the WOS license once the project license is selected;
5. keep the dependency out of `packages/wos-core/domain` unless it is genuinely part of a public domain contract and an ADR justifies it;
6. add tests covering the behavior for which the dependency was introduced;
7. update this file and `ROADMAP.md` in the same work.

Dependencies must not be added merely because they may be useful in a later wave.

Supported validation target: Linux amd64, PostgreSQL 18.6 and the SQLite version supplied by the pinned driver. Local browser acceptance on 2026-10-07 used Node 24.19.0, npm 11.9.0, Playwright 1.62.1 and system Chromium 151.0.7922.173. These are test tools, not production runtime dependencies.

Execution-contract digests use `github.com/cyberphone/json-canonicalization` pinned at `19d51d7fe467` (2024-12-13), Apache-2.0, RFC8785 implementation. Domain uses its pure canonicalizer, with UTF-16/property-order and numeric/string interoperability vectors. This is a direct dependency; no transport or storage imports in Domain.

The API-only wosctl client uses `go.yaml.in/yaml/v3 v3.0.5` (MIT/Apache-2.0) to parse syntax into bounded nodes. The WOS YAML 1.2 profile validates mappings, duplicate keys, depth/node/scalar/byte limits, decimal JSON numbers, strict booleans and unsupported syntax before typed decoding. YAML is not parsed in Domain; JCS remains the semantic digest format.
