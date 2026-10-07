# ADR-004 — Separate Issues and Blockers

Status: accepted, 2026-10-07.

An Issue records a problem. A Blocker records an explicit target, optional cause, propagation and release. Resolving an Issue does not release its Blockers; releasing one Blocker does not release the others. Read projections derive inherited blocking rather than changing descendant lifecycles.

Evidence: `application/issues_blockers.go; domain/blocker.go; inherited-blocker and compound-command contracts`.
