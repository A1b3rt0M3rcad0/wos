-- Namespace totals are queried only when required; Principal/Credential counts
-- keep using their narrower indexes. Cleanup scans only disposable completed
-- cache entries. No durable signed result or proof has a retention policy here.
CREATE INDEX signed_work_contracts_namespace_quota ON signed_work_contracts(namespace_id,status,expires_at);
CREATE INDEX work_review_contracts_namespace_quota ON work_review_contracts(namespace_id,status,expires_at);
CREATE INDEX idempotency_completed_expiry ON idempotency_records(expires_at,namespace_id,principal_id,command_name,idempotency_key) WHERE status='completed';
