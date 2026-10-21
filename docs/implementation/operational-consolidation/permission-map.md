# Legacy permission compatibility map

`legacy-v1` is a finite compatibility layer for grants and credential policies created before E15/E16 introduced command-specific permissions. It preserves the old permission strings and records an explicit compatibility marker on the Namespace grant and, when selected, the `CredentialPolicy`. It does not rewrite a permission list or infer authority from a permission name.

Each alias is keyed by the exact registered application command and the current capability required by that command. The mapping lives in [`application/permission_compatibility.go`](../../../packages/wos-core/application/permission_compatibility.go). Adding a command does not add a mapping. The authorization pipeline supplies the alias for that exact command; storage accepts it only when the corresponding grant and policy markers have been migrated.

## Frozen `legacy-v1` crosswalk

| Existing permission | Current capability | Exact command(s) |
| --- | --- | --- |
| `outcome:write` | `objective:create` | `CreateObjective`, `CreatePlanItems` |
| `outcome:write` | `objective:update` | `UpdateObjective`, `SetObjectiveOwners`, `SetObjectiveParent` |
| `outcome:write` | `objective:start` | `StartObjective` |
| `outcome:write` | `objective:cancel` | `CancelObjective` |
| `outcome:write` | `criterion:define` | `AddCriterion`, `ReviseCriterion`, `RetireCriterion`, `CreateOutcome`, `CreateObjective`, `CreateWorkItem`, `CreatePlanItems` |
| `outcome:write` | `dependency:write` | `AddDependency`, `RemoveDependency`, `ApplyDependencyChanges` |
| `work:write` | `task:create` | `CreateWorkItem`, `CreatePlanItems` |
| `work:write` | `task:update_definition` | `UpdateWorkItem`, `SetWorkItemExecutionSpec`, `SetWorkItemNotBefore`, `SetWorkItemObjective` |
| `work:write` | `task:activate` | `ActivateWorkItem` |
| `work:write` | `task:defer` | `DeferWorkItem` |
| `work:write` | `task:assign` | `SetWorkItemAssignees` |
| `work:write` | `task:claim` | `ClaimWorkItem`, `ReclaimWorkItem` |
| `work:write` | `task:renew` | `RenewWorkItemLease`, `RenewWorkContract`, `ResumeWorkContract`, `RenewSignedWorkContract`, `ResumeSignedWorkContract` |
| `work:write` | `task:release` | `ReleaseWorkItem` |
| `work:write` | `task:deliver` | `SubmitWorkResult`, `SyncWorkContract` |
| `work:write` | `task:accept` | `CompleteWorkItem` |
| `work:write` | `task:cancel` | `CancelWorkItem` |
| `work:write` | `task:reopen` | `ReopenWorkItem` |
| `work:write` | `issue:create` | `CreateIssue`, `ReportIssueWithBlocker`, `ReportIssueWithBlockers` |
| `work:write` | `issue:update` | `UpdateIssue`, `RelateIssueToWorkItem`, `RelateIssueWorkItem`, `RemoveIssueWorkItemRelation` |
| `work:write` | `issue:assign` | `AssignIssue`, `UnassignIssue` |
| `work:write` | `issue:investigate` | `InvestigateIssue`, `StartIssueInvestigation` |
| `work:write` | `issue:resolve` | `ResolveIssue`, `ResolveIssueAndBlockers`, `MarkIssueDuplicate`, `MarkIssueWontFix` |
| `work:write` | `issue:reopen` | `ReopenIssue` |
| `work:write` | `blocker:create` | `CreateBlocker`, `ReportIssueWithBlocker`, `ReportIssueWithBlockers` |
| `work:write` | `blocker:release` | `ResolveBlocker`, `ResolveIssueAndBlockers` |
| `work:write` | `blocker:cancel` | `CancelBlocker` |
| `work:write` | `records:write` | `AppendWorkActivity` |
| `work.contract.acquire` | `task:claim` | `AcquireSignedWorkContract`, `AcquireNextSignedWorkContract`, `AcquireWorkContract`, `AcquireNextWorkContract` |
| `work.contract.return` | `task:deliver` | `ReturnSignedWork` |
| `work.contract.complete_direct` | `task:accept` | `ReturnSignedWork` |
| `assessment:write` | `criterion:assess` | `AttestCriterion`, `RecordCriterionAssessment`, `ReturnSignedWork`, `ReturnSignedReview` |
| `assessment:waive` | `criterion:waive` | `RecordCriterionAssessment`, `ReturnSignedWork`, `ReturnSignedReview` |
| `work.review.acquire` | `task:review` | `AcquireSignedReviewContract`, `AcquireNextSignedReviewContract`, `RenewSignedReviewContract`, `ResumeSignedReviewContract` |
| `work.review.decide` | `task:review` | `ReturnSignedReview` |
| `work.review.decide` | `task:accept` | `ReturnSignedReview` |
| `work.review.decide` | `task:request_changes` | `ReturnSignedReview` |
| `conclusion:write` | `objective:conclude` | `AchieveObjective` |
| `conclusion:write` | `objective:reopen` | `ReopenObjective` |
| `conclusion:write` | `task:accept` | `FinalizeWorkContract` |
| `namespace:admin` | `policy:manage` | `SetNamespaceWorkProtocol`, `SetCredentialPolicy`, `SetAcceptancePolicy`, `SetReviewGroup` |
| `namespace:admin` | `task:intervene` | `InterveneSignedReviewCase`, `ReconcileExpiredWorkContracts` |

Composite commands still require the complete current command permission set. `CreatePlanItems` keeps its `planning:write` primary permission exactly; each created Objective/initial Criterion and Task additionally requires the historical `outcome:write` or `work:write` effect right. A grant with `work:write` cannot satisfy a policy that contains only `outcome:write` for an Objective/criterion operation, and vice versa.

## Rights intentionally not expanded

These remain exact permissions and receive no new aliases: signing key enrollment, rotation, and revocation; credential issue, update, revoke, and rotate; actor delegation; integration writes; work-contract revocation; administrator cancel/complete; and generic state, record, audit, or credential reads. Existing commands that still require `outcome:write`, `work:write`, `assessment:write`, or another unchanged capability continue to check that exact value.

The map grants no alias for commands added after the frozen crosswalk. In particular, a new `policy:manage` operation such as `ConfigureOutcomeOperationalProfile` cannot inherit `namespace:admin`. Unknown or future commands remain denied unless separately authorized by a later reviewed map version or an explicit granular grant.

## Preview, apply, and audit

`SecurityService.PreviewPermissionMigration` reads one coherent snapshot and reports the existing grant/policy, the current effective capabilities, and the set that `legacy-v1` would enable. It writes no state, version, audit row, or receipt. The per-target apply is an internal application method only; E17 does not publish it through HTTP, MCP, the capability catalogue, or CLI.

Apply requires `permissions:manage`, a reason, an expected active Namespace version, an explicit expected CredentialPolicy version when a credential is selected (zero represents an absent policy), and an idempotency key. SQLite/PostgreSQL guard the Namespace row and apply the grant marker, optional policy marker/revision, namespace revision, security audit row, and replay receipt atomically. A missing policy remains missing. Reapplying a completed intent returns the durable result; stale versions and concurrent revocation fail without mutation.

The grant's permission list and the policy's permitted operation list remain byte-for-byte the same values. The only policy change is its revision and the internal compatibility marker. Ordinary grant and CredentialPolicy commands cannot set the marker. Ordinary CredentialPolicy edits preserve an existing migrated marker, subject to their normal version guard.
