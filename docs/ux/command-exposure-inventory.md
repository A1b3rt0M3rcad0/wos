# Command exposure inventory

This is the explicit presentation contract, not authorization. Every command is checked against the executable catalog in Go tests. H: human journey; C: contextual; A: advanced administration; L: compatible legacy execution; S: signed external profile only; I: integration settings. A row does not make an action available in every state.

Workspace planning and documentary commands use the same authenticated Application pipeline through HTTP, MCP, the Go SDK and the restricted CLI planning surface. Batch planning requires `planning:write` plus the write permission for every created kind. New human defaults do not change older client defaults.

| Command | Class | Journey | Permission | Surface |
| --- | --- | --- | --- | --- |
| `abandon_outcome` | C | Outcome transitions | `outcome:write` | human |
| `accept_decision` | C | Decisions | `records:write` | human |
| `achieve_objective` | H | Objectives | `conclusion:write` | human |
| `achieve_outcome` | C | Outcome transitions | `conclusion:write` | human |
| `acquire_next_signed_review_contract` | S | Signed v2 review | `work.review.acquire` | profile |
| `acquire_next_signed_work_contract` | S | Signed v2 execution | `work.contract.acquire` | profile |
| `acquire_next_work_contract` | L | Legacy v1 contracts | `work.contract.acquire` | developer |
| `acquire_signed_review_contract` | S | Signed v2 review | `work.review.acquire` | profile |
| `acquire_signed_work_contract` | S | Signed v2 execution | `work.contract.acquire` | profile |
| `acquire_work_contract` | L | Legacy v1 contracts | `work.contract.acquire` | developer |
| `activate_outcome` | C | Outcome transitions | `outcome:write` | human |
| `activate_roadmap_revision` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `activate_work_item` | C | Task lifecycle | `work:write` | human |
| `add_criterion` | H | Success criteria | `outcome:write` | human |
| `add_dependency` | C | Dependencies | `outcome:write` | human |
| `administrative_cancel_work_item` | A | Task lifecycle | `work:admin_cancel` | developer |
| `administrative_complete_work_item` | A | Task lifecycle | `work:admin_complete` | developer |
| `append_entity_note` | C | Documentary updates | `records:write` | human |
| `append_work_activity` | S | Signed v2 execution activity | `work:write` | profile |
| `archive_outcome` | C | Outcome transitions | `outcome:write` | human |
| `archive_roadmap` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `attest_criterion` | H | Success criteria | `assessment:write` | developer |
| `cancel_blocker` | C | Issues and blockers | `work:write` | human |
| `cancel_objective` | H | Objectives | `outcome:write` | human |
| `cancel_work_item` | C | Task lifecycle | `work:write` | human |
| `claim_work_item` | L | Legacy v0 leases | `work:write` | human |
| `complete_work_item` | L | Legacy v0 leases | `work:write` | human |
| `configure_trigger` | I | Protocol and integrations | `integration:write` | developer |
| `create_blocker` | H | Issues and blockers | `work:write` | human |
| `create_evidence_link` | C | Evidence | `records:write` | human |
| `create_issue` | H | Issues and blockers | `work:write` | human |
| `create_objective` | H | Objectives | `outcome:write` | human |
| `create_outcome` | H | Outcome creation and editing | `outcome:write` | human |
| `create_plan_items` | H | Contextual planning batches | `planning:write` | human |
| `create_roadmap` | H | Roadmap creation and editing | `planning:write` | human |
| `create_work_item` | H | Task creation and editing | `work:write` | human |
| `deactivate_roadmap_revision` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `defer_work_item` | C | Task lifecycle | `work:write` | human |
| `discard_roadmap_draft` | C | Roadmap creation and editing | `planning:write` | human |
| `fail_outcome` | C | Outcome transitions | `outcome:write` | human |
| `finalize_work_contract` | L | Legacy v1 contracts | `conclusion:write` | developer |
| `intervene_signed_review_case` | S | Signed v2 intervention | `namespace:admin` | profile |
| `investigate_issue` | C | Issues and blockers | `work:write` | human |
| `link_external_reference` | A | External references | `outcome:write` | developer |
| `mark_issue_duplicate` | C | Issues and blockers | `work:write` | developer |
| `mark_issue_wont_fix` | C | Issues and blockers | `work:write` | developer |
| `open_roadmap_draft` | H | Roadmap creation and editing | `planning:write` | human |
| `propose_decision` | H | Decisions | `records:write` | human |
| `propose_decision_in_context` | C | Contextual decisions | `records:write` | human |
| `publish_roadmap_draft` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `reclaim_work_item` | L | Legacy v0 leases | `work:write` | developer |
| `reconcile_expired_work_contracts` | A | Legacy v1 contracts | `namespace:admin` | developer |
| `record_criterion_assessment` | H | Success criteria | `assessment:write` | human |
| `redeliver_delivery` | I | Protocol and integrations | `integration:write` | developer |
| `register_artifact` | H | Artifacts | `records:write` | human |
| `register_artifact_in_context` | C | Contextual artifacts | `records:write` | human |
| `register_evidence` | H | Evidence | `records:write` | human |
| `register_evidence_with_links` | H | Contextual evidence | `records:write` | human |
| `reject_decision` | C | Decisions | `records:write` | human |
| `release_work_item` | L | Legacy v0 leases | `work:write` | human |
| `remove_dependency` | H | Dependencies | `outcome:write` | human |
| `remove_external_reference` | A | External references | `outcome:write` | developer |
| `renew_signed_review_contract` | S | Signed v2 review | `work.review.acquire` | profile |
| `renew_signed_work_contract` | S | Signed v2 execution | `work:write` | profile |
| `renew_work_contract` | L | Legacy v1 contracts | `work:write` | developer |
| `renew_work_item_lease` | L | Legacy v0 leases | `work:write` | developer |
| `reopen_issue` | C | Issues and blockers | `work:write` | human |
| `reopen_objective` | H | Objectives | `conclusion:write` | human |
| `reopen_outcome` | C | Outcome transitions | `conclusion:write` | human |
| `reopen_roadmap` | C | Roadmap publication and lifecycle | `planning:write` | human |
| `reopen_work_item` | C | Task lifecycle | `work:write` | human |
| `replace_external_context` | A | External references | `outcome:write` | human |
| `replace_roadmap_draft` | H | Roadmap creation and editing | `planning:write` | human |
| `report_issue_with_blocker` | H | Issues and blockers | `work:write` | human |
| `resolve_blocker` | C | Issues and blockers | `work:write` | human |
| `resolve_issue` | C | Issues and blockers | `work:write` | human |
| `resolve_issue_and_blockers` | C | Issues and blockers | `work:write` | human |
| `resolve_signed_return_intent` | S | Explicit signed return resolution; current execution/review permission and original CID enforced in Core | `state:read` | profile |
| `resume_signed_review_contract` | S | Signed v2 review | `work.review.acquire` | profile |
| `resume_signed_work_contract` | S | Signed v2 execution | `work:write` | profile |
| `resume_work_contract` | L | Legacy v1 contracts | `work:write` | developer |
| `retire_criterion` | C | Success criteria | `outcome:write` | human |
| `retract_evidence` | C | Evidence | `records:write` | human |
| `retract_evidence_link` | C | Evidence | `records:write` | developer |
| `return_signed_review` | S | Signed v2 review | `work.review.decide` | profile |
| `return_signed_work` | S | Signed v2 execution | `work.contract.return` | profile |
| `revise_criterion` | C | Success criteria | `outcome:write` | human |
| `revoke_signed_contract` | S | Signed v2 intervention | `work.contract.revoke` | profile |
| `revoke_work_contract` | A | Legacy v1 contracts | `work.contract.revoke` | human |
| `set_namespace_work_protocol` | I | Protocol and integrations | `namespace:admin` | developer |
| `set_objective_owners` | C | Objectives | `outcome:write` | human |
| `set_objective_parent` | C | Objective placement | `outcome:write` | human |
| `set_outcome_owners` | C | Outcome participants | `outcome:write` | human |
| `set_trigger_enabled` | I | Protocol and integrations | `integration:write` | developer |
| `set_work_item_assignees` | C | Task creation and editing | `work:write` | human |
| `set_work_item_execution_spec` | C | Task specification | `work:write` | human |
| `set_work_item_not_before` | C | Task scheduling | `work:write` | human |
| `set_work_item_objective` | C | Task placement | `work:write` | human |
| `start_objective` | H | Objectives | `outcome:write` | human |
| `submit_work_result` | L | Legacy v1 contracts | `work:write` | developer |
| `supersede_decision` | C | Decisions | `records:write` | developer |
| `sync_work_contract` | L | Legacy v1 contracts | `work:write` | developer |
| `unarchive_outcome` | C | Outcome transitions | `outcome:write` | human |
| `update_blocker_description` | C | Issues and blockers | `work:write` | human |
| `update_decision` | C | Decisions | `records:write` | developer |
| `update_issue` | C | Issues and blockers | `work:write` | human |
| `update_objective` | C | Objectives | `outcome:write` | human |
| `update_outcome` | C | Outcome creation and editing | `outcome:write` | human |
| `update_trigger` | I | Protocol and integrations | `integration:write` | developer |
| `update_work_item` | C | Task creation and editing | `work:write` | human |
| `withdraw_artifact` | C | Artifacts | `records:write` | human |
