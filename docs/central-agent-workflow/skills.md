# Five official Skills

## Canonical source and packaging

The canonical bundles are `packages/wos-skill/skills/`. `packages/wos-skill/installer.go` embeds them into the native binary and owns install/update/doctor/removal; the Node installer remains a compatibility fixture. Installation supplies instructions, not credentials, server access, protocol activation, host permissions or an agent runtime.

| Skill              | Main responsibility                                 | Required operational path                                                 |
| ------------------ | --------------------------------------------------- | ------------------------------------------------------------------------- |
| `wos-workspace`    | Installation-scoped operation and Profile selection | init, profile list/use, explicit pin, CLI-owned records                   |
| `wos-coordination` | Discover/acquire/follow a WorkItem                  | available/next/context/status/check/finish                                |
| `wos-delegation`   | Auxiliary vs independent workers                    | pinned Profiles, one holder vs separate contracts, external host tools    |
| `wos-continuity`   | Resume without prior conversation                   | recover/context/status, bounded history and exact material references     |
| `wos-review`       | Independent exact-material evaluation               | reviewer Profile, review context/decision/assessment, reasoned correction |

## Rewrite procedure

1. Replace all instructions to open/read/interpret/edit internal contract documents with typed CLI operations. Remove direct MCP mutation examples from agent workflows; MCP remains a non-agent transport contract.
2. Remove per-project operational initialization. Keep runtime discovery paths such as `.agents/skills` distinct from installation data.
3. Put the compact happy path in each `SKILL.md`; keep references progressive and local to each installed bundle. A main file remains below the installer test's 6,500-character bound.
4. Centralize the wording of protected signing/recovery principles across self-contained references. Preserve exact-intention replay, prepared-only protection, signed receipt cleanup, issuer trust, expiry/fencing, material-bound review and correction acknowledgment.
5. Teach explicit `--profile`/`WOS_PROFILE` pinning for concurrent agents and unambiguous selected contract behavior.
6. Distinguish observations/activity from approval and local draft records from durable server registration. Do not claim history survives cleanup unless the underlying Server append succeeds.
7. Teach current context/status first; bounded history only for an unresolved question. Report truncation and cursors rather than interpreting omissions as absence.

## Tests and release gate

- Existing `packages/wos-skill/installer_test.go` verifies native install/update/remove, protected modified content and symlink guards.
- Existing `packages/wos-skill/test/installer.test.mjs` verifies portable frontmatter, length, progressive local references and compatibility fixtures.
- Replace obsolete direct-tool catalog assertions with a documentation contract check suited to CLI-only Skills: no direct MCP mutation names, no internal document edit instructions, required Profile pinning/bounded/recovery guidance.
- Install into a temporary supported runtime discovery root, run doctor, modify a copy, verify update/removal preserve user modifications.
- Execute the documented central CLI path against an actual server. File installation alone is not integration acceptance for Codex/Claude/Hermes/OpenClaw.

CAW-09 depends on settled command syntax and CAW-07 behavior. The implementation plan and main README must label source-branch behavior separately from the currently published binary/Skills.
