# Contributing

Start with [AGENTS.md](AGENTS.md), the development contract, and [ROADMAP.md](ROADMAP.md), the current implementation status. Read the relevant [architecture guidance](docs/architecture.md) and accepted [ADRs](docs/adr/) before changing domain behavior or public contracts.

## Set up and build

Use the Go version pinned in [.go-version](.go-version) (currently 1.27.2). Browser and distribution tooling also use Node.js and npm; the reviewed versions and dependencies are listed in [the dependency inventory](docs/dependencies.md). Go binaries do not require Node.js at runtime.

From the repository root:

```sh
go mod download
go build -o bin/wos ./packages/wos-api/cmd/wos
go build -o bin/wosctl ./packages/wos-cli/cmd/wosctl
./bin/wos version
```

See [service operations](docs/operations.md#local-installation) to run the service locally, or use the root Docker Compose configuration.

## Check your changes

Run the checks relevant to your change, including regression coverage for behavior changes:

```sh
go test ./...
go vet ./...
go test -race ./...
```

Format changed Go files with `gofmt`. The race detector requires a supported platform and C toolchain. PostgreSQL tests require a disposable PostgreSQL instance and `WOS_TEST_POSTGRES_DSN`; restore checks also need the matching database tools. Environment-dependent skips do not establish PostgreSQL, recovery, or native-platform acceptance. Never point tests at a database containing real work. [CI](.github/workflows/ci.yml) documents the complete verification environment and generated-contract checks.

For workspace changes, build both binaries above, then run:

```sh
cd tests/web
npm ci
npx playwright install --with-deps chromium
npm test
```

Playwright's browser dependencies may require system package installation. `npm run test:unit` runs only the browser unit checks; `npm test` also runs the real-server browser journeys. The [browser workflow](.github/workflows/web.yml) is the reproducible CI reference. Package and native-platform checks have their own [workflows](.github/workflows/).

## Send a focused pull request

- Describe the problem, resulting behavior, and validation, including skipped or unavailable checks.
- Keep changes small and preserve the separation between Core, transports, storage, and external agent execution.
- Update documentation, schemas, and ROADMAP entries when their contracts or status change. Write maintained product copy and documentation in English.
- Preserve owner-supplied artwork and historical evidence; follow the art instructions in AGENTS.md for visual work.
- Exclude credentials, private keys, local databases, and generated runtime state.

Use [GitHub Issues](https://github.com/A1b3rt0M3rcad0/wos/issues) for bugs and proposals. For suspected vulnerabilities, follow [SECURITY.md](SECURITY.md) instead of posting sensitive details publicly. Contributions are covered by the repository's [Apache-2.0 license](LICENSE).
