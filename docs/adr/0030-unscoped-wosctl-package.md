# ADR-030 — Public npm CLI identity

Status: accepted by explicit owner instruction, October 12, 2026.

The public CLI package is `wosctl`, installed with `npm install -g wosctl@latest`. This replaces ADR-029's historical `@woobeai/wos-cli` identity. Internal Go/package paths and the artifact kind `wos-cli` remain unchanged; the npm tarball is named `wosctl-VERSION.tgz`. Its native executables remain `wos` and `wosctl` on Linux/Windows amd64. Server runtime support remains Linux amd64.

Registry versions are immutable. Inspect existing metadata and compare integrity before upload. After a successful upload, bounded metadata polling accommodates propagation without repeating publication. Any conflicting integrity or authentication/network failure stops publication. A public installation receipt, not a local tarball alone, establishes consumer availability.

The owner further clarified that `wosctl` is the **only public npm package**. Native Go binaries embed the canonical skills; installation is `wos skill install --agent codex` (or another supported destination). Historical Node installers and package directories may remain compatibility/source fixtures, but are not built, versioned or published as separate service/skill npm packages. Native server/client archives and the OCI image remain separate deployment formats, not extra npm installations.

All documented operations use `wos <command>`, including profile/work/review and skills. The Go wos composition delegates API-client groups directly to the existing native client implementation, preserving its exit codes, CAS, credentials, signed receipts and context cancellation. The old wosctl executable is retained only as an explicit compatibility alias; it is not a second public installation or the instructed operation name.
