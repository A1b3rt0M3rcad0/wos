# WOS service

The existing Go service, distributed as a Linux amd64 npm package with an embedded native binary, HTTP/MCP and web workspace. Node >=22 runs the launcher; Core/server behavior stays in Go. No installation script downloads a binary or starts a process.

After registry publication:

```sh
npm install -g @a1b3rt0m3rcad0/wos@0.1.0
wos version
wos config validate
WOS_MCP_ENABLED=true wos server
```

Before npm publication, install the release tarball with `npm install -g /path/to/wos-0.1.0.tgz`. Projects can install locally and use `npx --no-install wos`. Keep credentials in the supported environment/secret facility; configure remote authentication, Namespace grants and persistent storage before accepting remote agents. The default local profile binds loopback and uses a trusted local identity. MCP stdio composes local storage; it is not an automatic remote bridge.

Native GitHub archives and GHCR images do not need Node. Service support is Linux amd64; `wos-skill` is portable. Do not infer platform certification from Go cross-compilation. Launch explicitly; npm installation does not deploy, migrate a live database, set up credentials or enable daemon management. Back up/restore and plan upgrades using the WOS operator documentation. Never run two SQLite instances against a live file without the supported coordination/storage policy.

Each package includes `release.json` with source commit, SemVer, source commit timestamp and binary SHA-256; launcher checks version/integrity before execution. Checksums detect corruption, not publisher authenticity; npm provenance and registry identity remain important. Reinstall rather than modifying embedded executables. The publisher supplies release-built content under `dist/npm/wos`; this source directory is not itself a ready-to-publish native package.

Docs: https://github.com/A1b3rt0M3rcad0/wos/blob/master/docs/operations.md

The npm/native/container distributions preserve WOS LICENSE plus Go and runtime-module license/notice files. `third-party-notices/index.md` identifies bundled dependency versions; test/build-only tools are not runtime package dependencies.

This package also provides `wosctl`, the API-only client, on Linux amd64. Both
executables identify the same version, commit and source timestamp and verify
embedded SHA-256 integrity before running. For example, `wosctl version --output json`
requires no workspace or running service. See [the client guide](https://github.com/A1b3rt0M3rcad0/wos/blob/master/packages/wos-cli/README.md)
for init, explicit acquisition, checkpoints, review and recovery. Native client
archives are separate for Linux amd64 and Windows amd64; cross-build alone is not
Windows acceptance. Installing this package does not start the service or an agent.

Generated editable workspace schemas are included in `schemas/`. Use the relevant section of `contract-v2.schema.json` for signed execution/review drafts; do not load its full technical envelope into agent context. Release metadata records signed v2/schema 2 and retained v1/schema 1 compatibility.
