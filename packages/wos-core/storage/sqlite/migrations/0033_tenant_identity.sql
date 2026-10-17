-- Tenant terminology cutover. Keep historical JSON and signed bytes unchanged.
-- Schema placement is provisioned separately; these IDs remain provenance guards.
ALTER TABLE aggregate_current_conclusions RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE artifacts RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE blockers RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE conclusion_assessments RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE conclusions RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE credential_policies RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE credentials RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE criterion_assessment_evidence RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE criterion_assessments RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE criterion_current_assessments RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE criterion_revisions RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE decisions RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE delivery_namespace_schedule RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE domain_events RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE entity_actor_links RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE entity_refs RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE evidence RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE evidence_links RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE idempotency_records RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE integration_deliveries RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE integration_events RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE issue_affected_refs RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE issues RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE namespace_grants RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE namespace_work_protocol RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE objectives RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE outcome_context_entries RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE outcome_coordination RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE outcome_external_references RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE outcomes RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE principal_review_groups RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE relations RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE roadmap_activation_history RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE roadmap_active_slots RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE roadmap_drafts RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE roadmap_revision_after_links RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE roadmap_revision_dependencies RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE roadmap_revision_nodes RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE roadmap_revisions RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE roadmaps RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE security_admin_results RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE security_audit RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE security_audit RENAME COLUMN namespace_version TO tenant_version;
ALTER TABLE signed_operation_results RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE signed_protocol_facts RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE signed_return_intent_resolutions RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE signed_work_contract_checkpoints RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE signed_work_contract_submissions RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE signed_work_contracts RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE signing_key_enrollments RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE signing_keys RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE signing_security_results RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE success_criteria RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE triggers RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE webhook_endpoints RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE work_acceptance_policies RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE work_activities RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE work_contract_checkpoints RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE work_contract_submissions RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE work_contracts RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE work_items RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE work_review_cases RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE work_review_contracts RENAME COLUMN namespace_id TO tenant_id;
ALTER TABLE delivery_namespace_schedule RENAME TO delivery_tenant_schedule;
ALTER TABLE namespace_grants RENAME TO tenant_grants;
ALTER TABLE namespace_work_protocol RENAME TO tenant_work_protocol;
ALTER TABLE namespaces RENAME TO tenants;

-- Mutable protocol metadata adopts a new writer epoch. The Go migration hook
-- checks old active work before this DDL and updates the current JSON record.
ALTER TABLE tenant_work_protocol RENAME TO tenant_work_protocol_before_tenant;
CREATE TABLE tenant_work_protocol (
 tenant_id TEXT PRIMARY KEY REFERENCES tenants(id),
 version INTEGER NOT NULL CHECK(version>0),
 phase TEXT NOT NULL CHECK(phase IN ('legacy','draining','contracts_v1','draining_to_signed_v2','signed_contracts_v2','draining_to_signed_v3','signed_contracts_v3')),
 writer_epoch INTEGER NOT NULL CHECK(writer_epoch IN (0,1,2,3)),
 state_json TEXT NOT NULL
);
INSERT INTO tenant_work_protocol(tenant_id,version,phase,writer_epoch,state_json)
 SELECT tenant_id,version,phase,writer_epoch,state_json FROM tenant_work_protocol_before_tenant;
DROP TABLE tenant_work_protocol_before_tenant;
