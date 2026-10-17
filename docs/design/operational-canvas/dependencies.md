# Dependency connections

The operational canvas uses the existing `AddDependency` and `RemoveDependency` commands. A connection runs from the dependent Objective or Task to its prerequisite within the same Outcome. It does not change hierarchy, Roadmap order, or lifecycle.

The **Dependency type** field defaults to **Hard prerequisite** (`hard`). Hard prerequisites participate in readiness and completion checks. **Advisory dependency** (`advisory`) records context without preventing a transition. Both strengths retain the server's self-reference, scope, endpoint, and cycle checks. A canvas-selected target initializes the normal typed reference picker and is resolved through the authorized reference query.

Following the owner's request for connection management, `remove_dependency` is now explicitly classified as **H / human** in the presentation inventory. Its authorization remains `outcome:write`; the browser gains no additional permission. The action is offered only for an active Relation whose `relation_type` is `depends_on`, never for a projected hierarchy, Roadmap link, Blocker, EvidenceLink, or another generic Relation type.

Removal requires a reason and submits the selected relation's ID and observed `expected_version` through the shared command/intention flow. The server enforces concurrency, source lifecycle, scope, and authentication. Removal marks the dependency as removed; it does not delete either endpoint or complete any work. An uncertain response retains its exact original command and idempotency key for recovery, and a version conflict requires an explicit fresh decision.
