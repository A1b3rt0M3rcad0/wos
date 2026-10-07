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
