# ADR-017 — Agent skills and coordinated distribution

- Status: Accepted
- Date: 2026-10-07
- Scope: extension to ADR-011; owner explicitly requested npm service/skills and automated releases.

## Decision

Keep `wos-api -> wos-core` unchanged. Add `packages/wos-npm` as a distribution wrapper for the existing Go service and `packages/wos-skill` as portable instructions plus a file installer. Neither owns domain behavior or an agent runtime. Agent creation, tool permissions, context isolation and filesystem execution remain consumer responsibilities.

Use a coordinated SemVer source in `VERSION`, scoped npm identities, a changes queue and a reviewable version PR. Publish only from verified `master` commits. Embed the supported native binary in the service tarball; no postinstall execution or remote binary download. Start with the accepted Linux amd64 matrix; portable skills do not imply native agent certification.

Skills guide explicit leases, fencing, versions, idempotency, bounded reads, proof and human review. Install into native discovery directories or an explicit generic directory without overwriting user changes. Credentials and MCP configuration remain outside package content.

## Consequences

Distribution versions cover both packages; Go module/API/protocol/schema versions remain distinct. Native consumers can use GitHub assets or GHCR without Node. npm installation is portable for skills but service support remains Linux amd64. Registry publication cannot be atomic; drafts, integrity checks and replayable jobs expose partial progress. Trusted publishing requires registry owner setup and working hosted CI. A tested implementation is not evidence of published packages or all-agent execution.

See [implementation plan](../distribution-plan.md) for acceptance, alternatives, tradeoffs and improvements.

## 2026-10-12 — Native WOS skill installation

The WOS Go executable also embeds the five canonical skill bundles from
`packages/wos-skill/skills` through a Go package located alongside their
npm distribution. `wos skill install|update|list|doctor|uninstall` provides
offline, Node-free native installation. It shares the npm installer's managed
file/hash format and safe preflight semantics; installation neither connects
MCP nor grants domain authority. The npm installer remains optional for
existing consumers and can inspect native-managed bundles without depending on
JSON map insertion order. This is a distribution/client feature, not a change
to Outcome/Task domain semantics.
