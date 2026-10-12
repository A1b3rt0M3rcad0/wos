# WOS releases and public installation

The only public npm package is **`wosctl`**:

```sh
npm install -g wosctl@latest
wos version
wos version --output json
wos skill install --agent codex
```

The package contains native Go `wos` and `wosctl` for Linux/Windows amd64. Node >=22 runs a small platform/integrity launcher; skill installation itself is native Go, offline and requires neither an API credential nor a running server. `all` covers Codex, Claude, Hermes and OpenClaw; `generic --dir` supports an explicit compatible discovery path. Installation does not start agents, configure MCP or grant execution authority. Server runtime support remains Linux amd64.

Native Linux server/client archives, the Windows client archive and the OCI image remain deployment formats. Historical separate service/skill npm directories are retained only as compatibility/source fixtures; they are not active public packages or publication targets.

## Prepare and validate

`VERSION` is the product SemVer. HTTP `/api/v1`, negotiated MCP protocol, `signed_contracts_v2`, YAML workspace schema and SQL migrations have independent compatibility identities. `.changes/` records bounded changes; `node tools/releases/version.mjs prepare` creates a reviewable version change, notes and release manifest. Preparation is not publication.

The workflow validates the exact original source with backend/PostgreSQL/race, browser, packages and native Linux/Windows CLI/keyring gates. Builds use pinned Go 1.27.2, Node 24.19.0 and npm 11.9.0; build timestamps derive from the source commit. `build.mjs`, `verify.mjs` and `reproducible.mjs` check source versions, SHA-256/SRI, offline installation, actual HTTP/MCP/UI, signed profiles/recovery, native skill operations, graceful shutdown, corruption rejection and byte identity across source paths.

Artifacts are `wosctl-VERSION.tgz`, `wos_VERSION_linux_amd64.tar.gz`, `wosctl_VERSION_linux_amd64.tar.gz`, `wosctl_VERSION_windows_amd64.zip`, `distribution.json` and `SHA256SUMS`. The OCI image is `ghcr.io/a1b3rt0m3rcad0/wos:VERSION`. Native receipts must match the actual platform, source, version and binary hashes; cross-compilation alone cannot certify Windows or a desktop keyring.

## Publish and reconcile

Initial 0.4.0 publication of `wosctl` is explicitly authorized and source-gated through `publish-cli-npm.yml`. After publication, Linux/Windows consumers install `wosctl@latest` directly from the public registry and exercise `wos skill` without a separately installed skills package. Publication and consumer receipts are uploaded as workflow artifacts.

Future coordinated publication uses explicit `workflow_dispatch` of `release.yml` with `publish=true`; `source_sha` can resume a full original integrated commit. It publishes only `wosctl` to npm, the versioned OCI image and the verified GitHub release assets. The default workflow prepares/verifies without publication. Registry credentials are configured in GitHub, never copied into source or chat. Private repositories cannot claim unsupported npm provenance.

Before any upload, compare existing registry integrity, tag/source identity and image labels. An existing byte-identical npm version is reconciled without upload. Different bytes stop immediately; immutable versions/tags/assets are never overwritten. After a successful upload, bounded polling tolerates temporary explicit 404 metadata propagation; authentication/network errors or a differing integrity stop immediately. Exhausted polling leaves publication unconfirmed and retains the original source/artifact for investigation. It never repeats an upload automatically.

Partial registry/image/asset failures leave the GitHub release unfinished. Resume with the original source; verify every existing artifact and finalize only after all required surfaces are proven. Never infer public availability from a prepared version or an upload process exit alone.

Current publication evidence and outstanding stable gates are tracked in [ROADMAP](../ROADMAP.md) and [the stable plan](stable/original-plan-2026-10-12.md). Apache-2.0 remains the effective license; bundled third-party notices travel with all native/npm/container artifacts.
