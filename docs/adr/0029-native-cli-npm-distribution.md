# ADR-029 — Native CLI npm distribution

- Status: Accepted for implementation in the 0.4.0 candidate
- Date: 2026-10-12
- Scope: Native `wos` and `wosctl` clients in npm; unchanged Domain/API storage boundaries

## Decision

Publish `@woobeai/wos-cli` as a separate npm package with exact-source Linux amd64 and Windows amd64 Go binaries. Expose `wos` and `wosctl` executable names, with a minimal Node launcher used exclusively to select the native host binary, validate source-bound checksums and relay exit status/signals. The actual skill installer and signed work client remain native Go implementations, without Node-powered logic or required server/network access for installing bundled Agent Skills.

Keep the Linux server npm package `@a1b3rt0m3rcad0/wos` and the optional pure instruction bundle `@a1b3rt0m3rcad0/wos-skill` separate. The three packages use coordinated VERSION and the same immutable release source. Do not install overlapping globally exposed `wos`/`wosctl` bins from both CLI and server npm distributions into one prefix.

Require actual native Linux and Windows host execution of Go `wos skill` install/doctor/uninstall and wosctl contracts, source/version hashes, offline npm package installation and the existing SQL/race/browser/package checks before publication. Only the initial native CLI npm package may be published separately on release 0.4.0. The full service/skills/GHCR release remains an explicitly authorized separate action.

Shipping `wos.exe` for Windows skill installation does not certify the Windows WOS server. Generating a cross-compiled executable or installing a skill file does not certify all third-party agents or their tools.

## Operational consequences

The initial CLI npm publisher is restricted to `VERSION=0.4.0` and `master` with an explicit npm token in GitHub Actions; it reconciles existing exact integrity rather than overwriting versions. Future versions continue under the coordinated, manually authorized release workflow. npm is a distribution/bootstrap dependency, not a requirement for `wos skill` command logic.
