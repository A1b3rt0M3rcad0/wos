# Implementation and review sequence

These are proposed review boundaries. The integrating agent may combine small tightly coupled changes, but must preserve runnable intermediate states and record actual verification before marking a unit complete. These titles do not authorize merge, release or registry publication.

| Order | Suggested commit                                                   | Contents                                                                                     | Depends on / review gate                                      |
| ----- | ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| 1     | `docs: define centralized agent workflow and authority boundaries` | Accepted ADR, original briefs, component plan, initial honest ROADMAP entry                  | Owner direction; no implementation claim                      |
| 2     | `feat(cli): centralize installation data and Profile selection`    | init, path resolution, active Profile, per-Profile server binding, pin precedence            | 1; existing protected file tests plus selection/process tests |
| 3     | `feat(core): persist scoped work activities and bounded history`   | Typed append/query, transactional persistence, storage contract, both SQL adapters           | 1; idempotency/rollback/order/auth tests                      |
| 4     | `feat(api): expose work activity and history across adapters`      | HTTP/MCP/SDK/OpenAPI/generation                                                              | 3; parity/public schema checks                                |
| 5     | `feat(cli): add bounded work discovery and contract context`       | available/next/context, exact/title/prefix resolution, omitted selector behavior             | 2, 4; ambiguity/incomplete-page tests                         |
| 6     | `feat(cli): record progress and delivery through typed commands`   | log, artifact/evidence, status/check, review/correction preparation                          | 2, 4, 5; protected draft/frozen-return checks                 |
| 7     | `feat(cli): import legacy local state without changing authority`  | Explicit dry-run migration, safe import, unchanged source/pending intent                     | 2; migration/crash/legacy regressions                         |
| 8     | `docs(skills): teach CLI-only coordination and continuation`       | Five Skills and self-contained references, installer/documentation contract tests            | 5, 6; exact syntax and actual installed behavior              |
| 9     | `test: verify central multiagent work and durable recovery`        | Real service acceptance, process pinning, independent review/correction, SQL restart/history | 2–8; actual source/platform/backend labels                    |
| 10    | `docs: publish central workflow guide and acceptance evidence`     | README/docs examples, source vs release note, validation, final ROADMAP and plan ZIP         | 9; links/commands/packages/required checks                    |

## Review checklist for every boundary

- Changes belong to the named capability; no unrelated visual/layout rewrite.
- Public commands, errors, fields and compatibility are documented with implementation.
- Server authority and protected request bytes are not weakened for convenience.
- Original source data and historical evidence remain traceable.
- Acceptance evidence belongs to the actual tested source and clearly identifies skips.
- README/Skills never imply the current published package includes unreleased branch commands.
- Final integration uses normal commits, retains reviewable history, and leaves merge/publication to the owner's explicit instruction.
