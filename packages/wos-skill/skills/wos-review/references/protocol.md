# Contract protocol and recovery

Check live capabilities and per-Namespace protocol before mutations. New contract commands use HTTP POST /api/v1/commands/{name} and MCP wos_{name} with {idempotency_key, command}; read the actual typed schema. Authenticated Principal governs access; Scope, ActorRef and external context cannot grant authority.

A WorkContract has one renewable lease. Only expired, revoked and completed are terminal. There is no release/unlock. Content CAS and lease CAS are distinct; fencing is a canonical positive decimal uint64 string. Renew before server expiry, never revive a terminated lease, and rotate execution/fencing only through explicit same-holder resume.

Snapshot digest is SHA256 of RFC8785 JCS over normalized typed spec, not YAML bytes. Submission digest binds exact canonical material IDs/revisions/checksums and criterion evidence. Checkpoints and evidence are progress/observations, not accepted completion. A revised submission requires material-bound review.

Save exact intent/key/destination before sending. After timeout recover its own receipt or replay the same payload. A retained receipt proves the original command, not current authority. If retention expired, examine bounded entities/history; absence is not proof of rollback. Stop on stale authority; compare conflicting bases rather than blindly updating expected_version.

Use bounded queries, explicit truncation/omission counts and cursors. Frozen specs can be cached after digest verification; live blockers, assessments and lease state cannot. Contracts do not lock files or make external side effects idempotent. The host coordinates those effects.

Legacy 0.1 commands remain only for unmigrated Namespaces. During draining new legacy claims/reclaims are rejected; after contracts_v1 cutover legacy execution mutations cannot create parallel leases or bypass contracts. Do not downgrade a migrated Namespace to work around errors.
