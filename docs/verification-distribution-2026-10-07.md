# Distribution verification — 2026-10-07

Implementation: D1–D4 in [ROADMAP](../ROADMAP.md), [plan](distribution-plan.md), ADR-017 and [release guide](releases.md). Changes are separated into planning, skills, native distribution, automated release and first version preparation PRs. These extend Waves 13/17/18; no Core/Application/HTTP/MCP mutation rule was changed.

## Executed locally

| Gate | Result |
| --- | --- |
| Go module verification / vet | Passed with Go 1.27.1 |
| `go test -json ./...` | 437 cases passed; zero failed/skipped tests; real PostgreSQL 18.6 configured; nine packages have no test files |
| `go test -race -json ./...` | 437 cases passed; zero failed/skipped tests |
| Browser | Three actual human/agent/workspace journeys passed in 20.9 s |
| Independent native service journey | All 18 steps passed over HTTP, including two agents, fencing, human proof, restart, clean restore and signed duplicate delivery |
| Node contracts | 15 cases passed: four-agent install/update/remove, edit/symlink preservation, portable references/live tool catalog, version synchronization, failure/replay, draft asset bytes and registry permission handling |
| Installed npm tarballs | Offline installation; real version/config/error status; four skill destinations; HTTP/MCP/UI readiness; SIGTERM propagation; corrupted binary rejection |
| Reproducibility | Native/npm archives and checksums byte-identical across separate source paths, same commit/toolchain |
| Versioned container | UID/GID 10001, read-only root, SQLite tmpfs, source/version labels, version/config, health, unauthorized denial, authenticated HTTP catalog/MCP, public runtime CA bundle |
| Workflows | actionlint 1.7.7 and YAML parsing passed; publish depends on all exact-source reusable gates |

Go suites, browser and journey exercise real WOS, not mock domain state. Publication recovery contracts use deterministic adapters and command fixtures; they do not prove external registry credentials, OIDC identity or a completed hosted release. Core/HTTP/MCP original acceptance remains intact.

Concise run outputs are preserved in `docs/audit/distribution-2026-10-07-*`. Intermediate artifact manifests identify their exact build commit; release assets must be rebuilt on the integrated release commit and pass the same consumer/reproducibility checks. Metadata is derived from source commit timestamp. Local build archives are not registry publication evidence.

## Scope and external blockers

Service support remains Linux amd64; skills are Node >=22 portable instruction bundles. Installation/discovery paths are verified against four upstream documented formats and exercised locally. No native run of every Claude/Codex/Hermes/OpenClaw version was performed; other agents require an Agent Skills reader or adapter. Skills do not launch runtimes or replace tool permissions/compaction.

Hosted Actions jobs cannot start because of failed account payments/spending limit, previously confirmed by job annotations. The current session has no authenticated npm publisher. Reading repository Actions Secrets returned 403; presence of a configured NPM_TOKEN cannot be inferred. Public `wos` is occupied by a different project. Use the proposed scoped identities after owner setup. Repositories/images retain their existing visibility; optional public-source provenance is disabled for this private source.

Implementation and local validation are complete; hosted execution, package ownership/credentials and final registry publication are external operational prerequisites. The automation retains all gates and never labels infrastructure failure as successful CI.
