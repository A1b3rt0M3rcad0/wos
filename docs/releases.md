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

`VERSION` is the product SemVer. HTTP `/api/v1`, negotiated MCP protocol, `signed_contracts_v2`, YAML workspace schema and SQL migrations have independent compatibility identities. `.changes/` records bounded changes; `node tools/releases/version.mjs prepare` creates the mechanical version change, notes and release manifest. On master, the preparation workflow validates and records only these files on a source-specific release branch. It creates a PR when the Actions bot is permitted; otherwise it records the prepared branch for an authorized agent to open the PR with its existing connection. It never bypasses master rules. Merging the version PR triggers the gated release. Application changes also use reviewed source PRs. Preparation is not publication.

The workflow validates the exact original source with backend/PostgreSQL/race, browser, packages and native Linux/Windows CLI/keyring gates. Builds use pinned Go 1.27.2, Node 24.19.0 and npm 11.9.0; build timestamps derive from the source commit. `build.mjs`, `verify.mjs` and `reproducible.mjs` check source versions, SHA-256/SRI, offline installation, actual HTTP/MCP/UI, signed profiles/recovery, native skill operations, graceful shutdown, corruption rejection and byte identity across source paths.

Artifacts are `wosctl-VERSION.tgz`, `wos_VERSION_linux_amd64.tar.gz`, `wosctl_VERSION_linux_amd64.tar.gz`, `wosctl_VERSION_windows_amd64.zip`, `distribution.json` and `SHA256SUMS`. The OCI image is `ghcr.io/a1b3rt0m3rcad0/wos:VERSION`. Native receipts must match the actual platform, source, version and binary hashes; cross-compilation alone cannot certify Windows or a desktop keyring.

## Publish and reconcile

Initial 0.4.0 publication of `wosctl` is explicitly authorized and source-gated through `publish-cli-npm.yml`. After publication, Linux/Windows consumers install `wosctl@latest` directly from the public registry and exercise `wos skill` without a separately installed skills package. Publication and consumer receipts are uploaded as workflow artifacts.

Merging a prepared release manifest into `master` automatically runs `release.yml`: exact-source tests → qualified artifacts → publication to npm, GHCR and GitHub Release. The publish job consumes the verified artifacts. Like WKS, publication is gated by workflow dependencies and does not require manual branch-rule setup. `workflow_dispatch` with `publish=true` and `source_sha` is the recovery path for the original integrated commit. Only `wosctl` is public on npm. Registry credentials are configured in GitHub, never copied into source or chat. Private repositories cannot claim unsupported npm provenance.

Before any upload, compare existing registry integrity, tag/source identity and image labels. An existing byte-identical npm version is reconciled without upload. Different bytes stop immediately; immutable versions/tags/assets are never overwritten. After a successful upload, bounded polling tolerates temporary explicit 404 metadata propagation; authentication/network errors or a differing integrity stop immediately. Exhausted polling leaves publication unconfirmed and retains the original source/artifact for investigation. It never repeats an upload automatically.

Partial registry/image/asset failures leave the GitHub release unfinished. Resume with the original source; verify every existing artifact and finalize only after all required surfaces are proven. Never infer public availability from a prepared version or an upload process exit alone.

Current publication evidence and outstanding stable gates are tracked in [ROADMAP](../ROADMAP.md) and [the stable plan](stable/original-plan-2026-10-12.md). Current source uses the [WOS Sustainable Use License 1.0](../LICENSE); earlier Apache-2.0 grants and upstream licenses remain effective for their material. Existing published versions and immutable artifacts are not relabeled or replaced. See [licensing](licensing.md).

New source distributions carry `LICENSE`, `TRADEMARKS.md`, `NOTICE.md`, `THIRD_PARTY_NOTICES.md`, `LICENSES/Apache-2.0.txt` and applicable generated upstream notices in npm, native archives and containers. Installed Skills retain the same legal files. npm metadata uses `SEE LICENSE IN LICENSE`; the OCI identifier is `LicenseRef-WOS-Sustainable-Use-1.0`. These changes do not publish a new version or alter immutable release checksums.


## Staged npm publication

An npm placeholder `0.0.0-stage` is not a released WOS package. If npm stages the upload, a package maintainer must approve the staged version with npm's 2FA before it becomes public; see [npm staged publishing](https://docs.npmjs.com/staged-publishing/). This is a registry requirement, not a WOS approval flow. Verify the stage against the original source and tarball; never rebuild/reupload the same immutable version. The workflow waits up to 15 minutes for the original version and preserves a pending diagnostic and artifacts when unavailable.

After approval, run `reconcile-npm.yml` with the original publication run ID. It reads that run's source gates, original package and native receipts, checks public SRI without uploading, then executes a registry-installed consumer on Linux/Windows. The consumer completes authenticated execution, independent review, correction and recovery outside the source checkout. A Windows test fixture does not add production Windows server support.
