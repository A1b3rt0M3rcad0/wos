# Command exposure inventory

This is the explicit presentation contract, not authorization. Every command is checked against the executable catalog in Go tests. H: human journey; C: contextual; A: advanced administration; L: compatible legacy execution; S: signed external profile only; I: integration settings. A row does not make an action available in every state.

| Command | Class | Journey |
| --- | --- | --- |
| `abandon_outcome` | C | Outcome — transições |
| `accept_decision` | C | Decisões |
| `achieve_objective` | H | Objectives |
| `achieve_outcome` | C | Outcome — transições |
| `acquire_next_signed_review_contract` | S | Contracts v2 assinados — revisão |
| `acquire_next_signed_work_contract` | S | Contracts v2 assinados — execução |
| `acquire_next_work_contract` | L | Contracts v1 |
| `acquire_signed_review_contract` | S | Contracts v2 assinados — revisão |
| `acquire_signed_work_contract` | S | Contracts v2 assinados — execução |
| `acquire_work_contract` | L | Contracts v1 |
| `activate_outcome` | C | Outcome — transições |
| `activate_roadmap_revision` | C | Roadmaps — publicação/estado |
| `activate_work_item` | C | Tasks — estado |
| `add_criterion` | H | Critérios |
| `add_dependency` | C | Dependências |
| `administrative_cancel_work_item` | A | Tasks — estado |
| `administrative_complete_work_item` | A | Tasks — estado |
| `archive_outcome` | C | Outcome — transições |
| `archive_roadmap` | C | Roadmaps — publicação/estado |
| `attest_criterion` | H | Critérios |
| `cancel_blocker` | C | Issues/Blockers |
| `cancel_objective` | H | Objectives |
| `cancel_work_item` | C | Tasks — estado |
| `claim_work_item` | L | Lease legacy v0 |
| `complete_work_item` | L | Lease legacy v0 |
| `configure_trigger` | I | Protocolo/integração |
| `create_blocker` | H | Issues/Blockers |
| `create_evidence_link` | C | Evidências |
| `create_issue` | H | Issues/Blockers |
| `create_objective` | H | Objectives |
| `create_outcome` | H | Outcome — criação/edição |
| `create_roadmap` | H | Roadmaps — criação e edição |
| `create_work_item` | H | Tasks — criação/edição |
| `deactivate_roadmap_revision` | C | Roadmaps — publicação/estado |
| `defer_work_item` | C | Tasks — estado |
| `discard_roadmap_draft` | C | Roadmaps — criação e edição |
| `fail_outcome` | C | Outcome — transições |
| `finalize_work_contract` | L | Contracts v1 |
| `intervene_signed_review_case` | S | Contracts v2 assinados — intervenção |
| `investigate_issue` | C | Issues/Blockers |
| `link_external_reference` | A | Referências externas |
| `mark_issue_duplicate` | C | Issues/Blockers |
| `mark_issue_wont_fix` | C | Issues/Blockers |
| `open_roadmap_draft` | H | Roadmaps — criação e edição |
| `propose_decision` | H | Decisões |
| `publish_roadmap_draft` | C | Roadmaps — publicação/estado |
| `reclaim_work_item` | L | Lease legacy v0 |
| `reconcile_expired_work_contracts` | A | Contracts v1 |
| `record_criterion_assessment` | H | Critérios |
| `redeliver_delivery` | I | Protocolo/integração |
| `register_artifact` | H | Artefatos |
| `register_evidence` | H | Evidências |
| `reject_decision` | C | Decisões |
| `release_work_item` | L | Lease legacy v0 |
| `remove_dependency` | C | Dependências |
| `remove_external_reference` | A | Referências externas |
| `renew_signed_review_contract` | S | Contracts v2 assinados — revisão |
| `renew_signed_work_contract` | S | Contracts v2 assinados — execução |
| `renew_work_contract` | L | Contracts v1 |
| `renew_work_item_lease` | L | Lease legacy v0 |
| `reopen_issue` | C | Issues/Blockers |
| `reopen_objective` | H | Objectives |
| `reopen_outcome` | C | Outcome — transições |
| `reopen_roadmap` | C | Roadmaps — publicação/estado |
| `reopen_work_item` | C | Tasks — estado |
| `replace_external_context` | A | Referências externas |
| `replace_roadmap_draft` | H | Roadmaps — criação e edição |
| `report_issue_with_blocker` | H | Issues/Blockers |
| `resolve_blocker` | C | Issues/Blockers |
| `resolve_issue` | C | Issues/Blockers |
| `resolve_issue_and_blockers` | C | Issues/Blockers |
| `resume_signed_review_contract` | S | Contracts v2 assinados — revisão |
| `resume_signed_work_contract` | S | Contracts v2 assinados — execução |
| `resume_work_contract` | L | Contracts v1 |
| `retire_criterion` | C | Critérios |
| `retract_evidence` | C | Evidências |
| `retract_evidence_link` | C | Evidências |
| `return_signed_review` | S | Contracts v2 assinados — revisão |
| `return_signed_work` | S | Contracts v2 assinados — execução |
| `revise_criterion` | C | Critérios |
| `revoke_signed_contract` | S | Contracts v2 assinados — intervenção |
| `revoke_work_contract` | A | Contracts v1 |
| `set_namespace_work_protocol` | I | Protocolo/integração |
| `set_objective_owners` | A | Objectives |
| `set_outcome_owners` | A | Outcome — participantes |
| `set_trigger_enabled` | I | Protocolo/integração |
| `set_work_item_assignees` | A | Tasks — criação/edição |
| `start_objective` | H | Objectives |
| `submit_work_result` | L | Contracts v1 |
| `supersede_decision` | C | Decisões |
| `sync_work_contract` | L | Contracts v1 |
| `unarchive_outcome` | C | Outcome — transições |
| `update_blocker_description` | C | Issues/Blockers |
| `update_decision` | C | Decisões |
| `update_issue` | C | Issues/Blockers |
| `update_objective` | C | Objectives |
| `update_outcome` | C | Outcome — criação/edição |
| `update_trigger` | I | Protocolo/integração |
| `update_work_item` | C | Tasks — criação/edição |
| `withdraw_artifact` | C | Artefatos |
