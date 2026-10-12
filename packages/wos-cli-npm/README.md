# WOS native CLI

This npm distribution installs the **native Go** `wos` and `wosctl` executables on Linux x64 and Windows x64. npm/Node.js is used only for package installation and the small platform/integrity launcher. The commands themselves execute compiled Go code. In particular, `wos skill install` runs a Go installer that embeds the canonical WOS Agent Skills and needs no Node.js logic, network access or running WOS server.

```sh
npm install -g wosctl@latest
wos version
wos skill install --agent codex
wos skill install --agent claude
wos skill install --agent hermes
wos skill install --agent openclaw
wos skill install --agent all
wos skill install --agent generic --dir /path/to/skills
wos version --output json
```

Use `wos skill doctor --agent codex`, `wos skill update --agent codex`, or `wos skill uninstall --agent codex` to manage installed skills. `--global`, `--project` and `--dry-run` are supported. The installer preserves user-modified/unmanaged skill directories and does not provision tokens, MCP connections or agent tool permissions.

`wosctl` is the API-only remote work client; it does not run a WOS server. Its signed workspace flow requires independently provisioned authorization and issuer trust. Consult the root WOS README and `packages/wos-cli/README.md` for supported protocols.

**Platform scope:** Linux amd64 and Windows amd64 clients and skill installation. The WOS server itself is certified only for Linux amd64, and shipping `wos.exe` for its `skill` subcommand must not be interpreted as Windows server certification.

The executable checks source-bound SHA-256 metadata at each invocation. No package install script downloads a binary or starts a service. npm versions are coordinated with the WOS source version; package names and CLI commands have separate identities. `wosctl` is the only public npm package. All operations use `wos <command>`; the older `wosctl` executable is retained as a compatibility alias. There is no separately installed service or skills npm package.
