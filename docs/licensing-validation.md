# Licensing propagation verification

Executed on 2026-10-17 for the owner's source-available licensing change. This record qualifies software metadata, notice propagation and safe Skill installation; it is not a legal opinion or a release publication receipt.

## Executed scope

| Check | Result |
| --- | --- |
| Policy consistency | Independent read-only comparison against the pinned Woobe reference and WOS's README, licensing guide, compatibility policy and ADR 037 found no material contradictions. Internal production and own commercial applications/OaaS remain permitted; commercial platform supply is excluded; prior Apache and third-party grants remain preserved. |
| Focused Node tests | All 30 checks pass across Skills installation, distribution metadata/integrity, native-Go test support and release compatibility. Missing or modified legal files in actual archives are rejected. |
| Affected Go packages | `go test -race -count=1 ./packages/wos-skill ./packages/wos-api/cmd/wos` passes. Installer tests retain managed-file protections, upgrade intact pre-notice bundles and preserve user-edited notices. |
| Actual distribution | Build and verify the npm tarball, Linux server archive, Linux CLI archive and Windows CLI ZIP. Read their actual legal-file members and compare with root bytes. Offline-installed npm metadata and legal files match source. |
| Installed Skills | The new native binary installs all five Skills into four discovery layouts. All 20 installations carry matching legal files, and existing install/update/doctor/uninstall safeguards pass. The JavaScript compatibility installer is covered by its focused tests. |
| Packaged functionality | Complete the verifier's actual HTTP/MCP/UI readiness, signed checkout/delivery/independent review/recovery, graceful shutdown and deliberate checksum-corruption checks. |
| Source notices | Preserve the prior Apache text byte for byte and the original dependency/font license files. Current first-party package copies match the root license and notices. |
| Source/release boundary | npm identifies `SEE LICENSE IN LICENSE`; OCI uses `LicenseRef-WOS-Sustainable-Use-1.0`. The read-only release plan recognizes the major compatibility change. No version, schema fingerprint, immutable artifact or publication manifest is rewritten. |
| Documentation and README | Local paths and incoming README anchors pass. GitHub Markdown API output, including the updated license badge, is inspected in Chromium at 1280, 390 and 320 CSS pixels; all eight images decode and both detail sections expand without page-level horizontal overflow. |

The initial builds tested the new working tree while native build metadata still identified its previous HEAD, `8e2e83b`. They are working-tree qualification, not exact-head release evidence. Exact committed-source checks and subsequent results are reported by [PR #111](https://github.com/A1b3rt0M3rcad0/wos/pull/111/checks).

## Reproduce

Use the existing pinned Node/Go environment, install the repository's existing development prerequisites, and keep caches in writable locations. This environment requires a writable `XDG_CACHE_HOME` for the packaged-client smoke run.

```sh
node --test packages/wos-skill/test/*.test.mjs tools/distribution/test.mjs tools/distribution/native-go-tests.test.mjs tools/releases/test.mjs tools/releases/compatibility.test.mjs
go test -race -count=1 ./packages/wos-skill ./packages/wos-api/cmd/wos
node tools/releases/version.mjs plan
node tools/distribution/build.mjs
node tools/distribution/verify.mjs
```

After committing a clean source tree, build once more and run `node tools/distribution/reproducible.mjs` for byte identity across separate source paths. Preparation, publication and legal review remain separate activities.

## Limits

The Windows ZIP is cross-built and inspected locally; that is not Windows execution qualification. Container labels, copy paths and notice checks are implemented, but a Docker engine was not exercised in this local verification. Hosted platform/container checks report their own results. Full domain/database suites are not rerun locally for this licensing change. No deployed service, registry, earlier release or stored user authority is modified.
