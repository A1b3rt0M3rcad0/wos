# Command exposure inventory

This is the explicit presentation contract, not authorization. Every command is checked against the executable catalog in Go tests. H: human journey; C: contextual; A: advanced administration; L: compatible legacy execution; S: signed external profile only; I: integration settings. A row does not make an action available in every state.

Workspace planning and documentary commands use the same authenticated Application pipeline through HTTP, MCP, the Go SDK and the restricted CLI planning surface. Batch planning requires `planning:write` plus the write permission for every created kind. New human defaults do not change older client defaults.

| Command | Class | Journey | Permission | Surface |
| --- | --- | --- | --- | --- |
| `abandon_outcome` | C | Outcome transitions | `outcome:write` | human |
| `accept_decision` | C | Decisions | `records:write` | human |
| `achieve_objective` | H | Objectives | `objective:conclude` | human |
| `achieve_outcome` | C | Outcome transitions | `conclusion:write` | human |
| `acquire_next_signed_review_contract` | S | Signed v2 review | `task:review` | profile |
| `acquire_next_signed_work_contract` | S | Signed v2 execution | `task:claim` | profile |
| `acquire_next_work_contract` | L | Legacy v1 contracts | `task:claim` | developer |
| `acquire_signed_review_contract` | S | Signed v2 review | `task:review` | profile |
| `acquire_signed_work_contract` | S | Signed v2 execution | `task:claim` | profile |
| `acquire_work_contract` | L | Legacy v1 contracts | `task:claim` | developer |
| `activate_outcome` | C | Outcome transitions | `outcome:write` | human |
| `activate_roadmap_revision` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `activate_work_item` | C | Task lifecycle | `task:activate` | human |
| `add_criterion` | H | Success criteria | `criterion:define` | human |
| `add_dependency` | C | Dependencies | `dependency:write` | human |
| `administrative_cancel_work_item` | A | Task lifecycle | `work:admin_cancel` | developer |
| `administrative_complete_work_item` | A | Task lifecycle | `work:admin_complete` | developer |
| `append_entity_note` | C | Documentary updates | `records:write` | human |
| `append_work_activity` | S | Signed v2 execution activity | `records:write` | profile |
| `apply_dependency_changes` | C | Dependencies | `dependency:write` | human |
| `archive_outcome` | C | Outcome transitions | `outcome:write` | human |
| `archive_roadmap` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `assign_issue` | H | Issues and blockers | `issue:assign` | human |
| `attest_criterion` | H | Success criteria | `criterion:assess` | developer |
| `cancel_blocker` | C | Issues and blockers | `blocker:cancel` | human |
| `cancel_objective` | H | Objectives | `objective:cancel` | human |
| `cancel_work_item` | C | Task lifecycle | `task:cancel` | human |
| `claim_work_item` | L | Legacy v0 leases | `task:claim` | human |
| `complete_work_item` | L | Legacy v0 leases | `task:accept` | human |
| `configure_outcome_operational_profile` | A | Operational profile migration | `policy:manage` | developer |
| `configure_trigger` | I | Protocol and integrations | `integration:write` | developer |
| `create_blocker` | H | Issues and blockers | `blocker:create` | human |
| `create_evidence_link` | C | Evidence | `records:write` | human |
| `create_issue` | H | Issues and blockers | `issue:create` | human |
| `create_objective` | H | Objectives | `objective:create` | human |
| `create_outcome` | H | Outcome creation and editing | `outcome:write` | human |
| `create_plan_items` | H | Contextual planning batches | `planning:write` | human |
| `create_roadmap` | H | Roadmap creation and editing | `planning:write` | human |
| `create_work_item` | H | Task creation and editing | `task:create` | human |
| `deactivate_roadmap_revision` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `defer_work_item` | C | Task lifecycle | `task:defer` | human |
| `discard_roadmap_draft` | C | Roadmap creation and editing | `planning:write` | human |
| `fail_outcome` | C | Outcome transitions | `outcome:write` | human |
| `finalize_work_contract` | L | Legacy v1 contracts | `task:accept` | developer |
| `intervene_signed_review_case` | S | Signed v2 intervention | `task:intervene` | profile |
| `investigate_issue` | C | Issues and blockers | `issue:investigate` | human |
| `link_external_reference` | A | External references | `outcome:write` | developer |
| `mark_issue_duplicate` | C | Issues and blockers | `issue:resolve` | developer |
| `mark_issue_wont_fix` | C | Issues and blockers | `issue:resolve` | developer |
| `open_roadmap_draft` | H | Roadmap creation and editing | `planning:write` | human |
| `propose_decision` | H | Decisions | `records:write` | human |
| `propose_decision_in_context` | C | Contextual decisions | `records:write` | human |
| `publish_roadmap_draft` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `reclaim_work_item` | L | Legacy v0 leases | `task:claim` | human |
| `reconcile_expired_work_contracts` | A | Legacy v1 contracts | `task:intervene` | developer |
| `record_criterion_assessment` | H | Success criteria | `criterion:assess` | human |
| `redeliver_delivery` | I | Protocol and integrations | `integration:write` | developer |
| `register_artifact` | H | Artifacts | `records:write` | human |
| `register_artifact_in_context` | C | Contextual artifacts | `records:write` | human |
| `register_evidence` | H | Evidence | `records:write` | human |
| `register_evidence_with_links` | H | Contextual evidence | `records:write` | human |
| `reject_decision` | C | Decisions | `records:write` | human |
| `relate_issue_to_work_item` | C | Issues and blockers | `issue:update` | human |
| `release_work_item` | L | Legacy v0 leases | `task:release` | human |
| `remove_dependency` | H | Dependencies | `dependency:write` | human |
| `remove_external_reference` | A | External references | `outcome:write` | developer |
| `remove_issue_work_item_relation` | C | Issues and blockers | `issue:update` | human |
| `renew_signed_review_contract` | S | Signed v2 review | `task:review` | profile |
| `renew_signed_work_contract` | S | Signed v2 execution | `task:renew` | profile |
| `renew_work_contract` | L | Legacy v1 contracts | `task:renew` | developer |
| `renew_work_item_lease` | L | Legacy v0 leases | `task:renew` | human |
| `reopen_issue` | C | Issues and blockers | `issue:reopen` | human |
| `reopen_objective` | H | Objectives | `objective:reopen` | human |
| `reopen_outcome` | C | Outcome transitions | `conclusion:write` | human |
| `reopen_roadmap` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `reopen_work_item` | C | Task lifecycle | `task:reopen` | human |
| `replace_external_context` | A | External references | `outcome:write` | human |
| `replace_roadmap_draft` | H | Roadmap creation and editing | `planning:write` | human |
| `report_issue_with_blocker` | H | Issues and blockers | `issue:create` | human |
| `report_issue_with_blockers` | H | Issues and blockers | `issue:create` | human |
| `resolve_blocker` | C | Issues and blockers | `blocker:release` | human |
| `resolve_issue` | C | Issues and blockers | `issue:resolve` | human |
| `resolve_issue_and_blockers` | C | Issues and blockers | `issue:resolve` | human |
| `resolve_signed_return_intent` | S | Explicit signed return resolution; current execution/review permission and original CID enforced in Core | `state:read` | profile |
| `resume_signed_review_contract` | S | Signed v2 review | `task:review` | profile |
| `resume_signed_work_contract` | S | Signed v2 execution | `task:renew` | profile |
| `resume_work_contract` | L | Legacy v1 contracts | `task:renew` | developer |
| `retire_criterion` | C | Success criteria | `criterion:define` | human |
| `retract_evidence` | C | Evidence | `records:write` | human |
| `retract_evidence_link` | C | Evidence | `records:write` | developer |
| `return_signed_review` | S | Signed v2 review | `task:review` | profile |
| `return_signed_work` | S | Signed v2 execution | `task:deliver` | profile |
| `revise_criterion` | C | Success criteria | `criterion:define` | human |
| `revoke_signed_contract` | S | Signed v2 intervention | `work.contract.revoke` | profile |
| `revoke_work_contract` | A | Legacy v1 contracts | `work.contract.revoke` | human |
| `set_namespace_work_protocol` | I | Protocol and integrations | `policy:manage` | developer |
| `set_objective_owners` | C | Objectives | `objective:update` | human |
| `set_objective_parent` | C | Objective placement | `objective:update` | human |
| `set_outcome_owners` | C | Outcome participants | `outcome:write` | human |
| `set_trigger_enabled` | I | Protocol and integrations | `integration:write` | developer |
| `set_work_item_assignees` | C | Task creation and editing | `task:assign` | human |
| `set_work_item_execution_spec` | C | Task specification | `task:update_definition` | human |
| `set_work_item_not_before` | C | Task scheduling | `task:update_definition` | human |
| `set_work_item_objective` | C | Task placement | `task:update_definition` | human |
| `start_issue_investigation` | C | Issues and blockers | `issue:investigate` | human |
| `start_objective` | H | Objectives | `objective:start` | human |
| `submit_work_result` | L | Legacy v1 contracts | `task:deliver` | developer |
| `supersede_decision` | C | Decisions | `records:write` | developer |
| `sync_work_contract` | L | Legacy v1 contracts | `task:deliver` | developer |
| `unarchive_outcome` | C | Outcome transitions | `outcome:write` | human |
| `unassign_issue` | C | Issues and blockers | `issue:assign` | human |
| `update_blocker_description` | C | Issues and blockers | `work:write` | human |
| `update_decision` | C | Decisions | `records:write` | developer |
| `update_issue` | C | Issues and blockers | `issue:update` | human |
| `update_objective` | C | Objectives | `objective:update` | human |
| `update_outcome` | C | Outcome creation and editing | `outcome:write` | human |
| `update_trigger` | I | Protocol and integrations | `integration:write` | developer |
| `update_work_item` | C | Task creation and editing | `task:update_definition` | human |
| `withdraw_artifact` | C | Artifacts | `records:write` | human |
