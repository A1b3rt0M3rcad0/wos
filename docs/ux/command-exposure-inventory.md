# Command exposure inventory

This is the explicit presentation contract, not authorization. Every command is checked against the executable catalog in Go tests. H: human journey; C: contextual; A: advanced administration; L: compatible legacy execution; S: signed external profile only; I: integration settings. A row does not make an action available in every state.

| Command | Class | Journey |
| --- | --- | --- |
| `abandon_outcome` | C | Outcome transitions |
| `accept_decision` | C | Decisions |
| `achieve_objective` | H | Objectives |
| `achieve_outcome` | C | Outcome transitions |
| `acquire_next_signed_review_contract` | S | Signed v2 review |
| `acquire_next_signed_work_contract` | S | Signed v2 execution |
| `acquire_next_work_contract` | L | Legacy v1 contracts |
| `acquire_signed_review_contract` | S | Signed v2 review |
| `acquire_signed_work_contract` | S | Signed v2 execution |
| `acquire_work_contract` | L | Legacy v1 contracts |
| `activate_outcome` | C | Outcome transitions |
| `activate_roadmap_revision` | C | Roadmap publication and lifecycle |
| `activate_work_item` | C | Task lifecycle |
| `add_criterion` | H | Success criteria |
| `add_dependency` | C | Dependencies |
| `administrative_cancel_work_item` | A | Task lifecycle |
| `administrative_complete_work_item` | A | Task lifecycle |
| `archive_outcome` | C | Outcome transitions |
| `archive_roadmap` | C | Roadmap publication and lifecycle |
| `attest_criterion` | H | Success criteria |
| `cancel_blocker` | C | Issues and blockers |
| `cancel_objective` | H | Objectives |
| `cancel_work_item` | C | Task lifecycle |
| `claim_work_item` | L | Legacy v0 leases |
| `complete_work_item` | L | Legacy v0 leases |
| `configure_trigger` | I | Protocol and integrations |
| `create_blocker` | H | Issues and blockers |
| `create_evidence_link` | C | Evidence |
| `create_issue` | H | Issues and blockers |
| `create_objective` | H | Objectives |
| `create_outcome` | H | Outcome creation and editing |
| `create_roadmap` | H | Roadmap creation and editing |
| `create_work_item` | H | Task creation and editing |
| `deactivate_roadmap_revision` | C | Roadmap publication and lifecycle |
| `defer_work_item` | C | Task lifecycle |
| `discard_roadmap_draft` | C | Roadmap creation and editing |
| `fail_outcome` | C | Outcome transitions |
| `finalize_work_contract` | L | Legacy v1 contracts |
| `intervene_signed_review_case` | S | Signed v2 intervention |
| `investigate_issue` | C | Issues and blockers |
| `link_external_reference` | A | External references |
| `mark_issue_duplicate` | C | Issues and blockers |
| `mark_issue_wont_fix` | C | Issues and blockers |
| `open_roadmap_draft` | H | Roadmap creation and editing |
| `propose_decision` | H | Decisions |
| `publish_roadmap_draft` | C | Roadmap publication and lifecycle |
| `reclaim_work_item` | L | Legacy v0 leases |
| `reconcile_expired_work_contracts` | A | Legacy v1 contracts |
| `record_criterion_assessment` | H | Success criteria |
| `redeliver_delivery` | I | Protocol and integrations |
| `register_artifact` | H | Artifacts |
| `register_evidence` | H | Evidence |
| `reject_decision` | C | Decisions |
| `release_work_item` | L | Legacy v0 leases |
| `remove_dependency` | C | Dependencies |
| `remove_external_reference` | A | External references |
| `renew_signed_review_contract` | S | Signed v2 review |
| `renew_signed_work_contract` | S | Signed v2 execution |
| `renew_work_contract` | L | Legacy v1 contracts |
| `renew_work_item_lease` | L | Legacy v0 leases |
| `reopen_issue` | C | Issues and blockers |
| `reopen_objective` | H | Objectives |
| `reopen_outcome` | C | Outcome transitions |
| `reopen_roadmap` | C | Roadmap publication and lifecycle |
| `reopen_work_item` | C | Task lifecycle |
| `replace_external_context` | A | External references |
| `replace_roadmap_draft` | H | Roadmap creation and editing |
| `report_issue_with_blocker` | H | Issues and blockers |
| `resolve_blocker` | C | Issues and blockers |
| `resolve_issue` | C | Issues and blockers |
| `resolve_issue_and_blockers` | C | Issues and blockers |
| `resume_signed_review_contract` | S | Signed v2 review |
| `resume_signed_work_contract` | S | Signed v2 execution |
| `resume_work_contract` | L | Legacy v1 contracts |
| `retire_criterion` | C | Success criteria |
| `retract_evidence` | C | Evidence |
| `retract_evidence_link` | C | Evidence |
| `return_signed_review` | S | Signed v2 review |
| `return_signed_work` | S | Signed v2 execution |
| `revise_criterion` | C | Success criteria |
| `revoke_signed_contract` | S | Signed v2 intervention |
| `revoke_work_contract` | A | Legacy v1 contracts |
| `set_namespace_work_protocol` | I | Protocol and integrations |
| `set_objective_owners` | A | Objectives |
| `set_outcome_owners` | A | Outcome participants |
| `set_trigger_enabled` | I | Protocol and integrations |
| `set_work_item_assignees` | A | Task creation and editing |
| `start_objective` | H | Objectives |
| `submit_work_result` | L | Legacy v1 contracts |
| `supersede_decision` | C | Decisions |
| `sync_work_contract` | L | Legacy v1 contracts |
| `unarchive_outcome` | C | Outcome transitions |
| `update_blocker_description` | C | Issues and blockers |
| `update_decision` | C | Decisions |
| `update_issue` | C | Issues and blockers |
| `update_objective` | C | Objectives |
| `update_outcome` | C | Outcome creation and editing |
| `update_trigger` | I | Protocol and integrations |
| `update_work_item` | C | Task creation and editing |
| `withdraw_artifact` | C | Artifacts |
