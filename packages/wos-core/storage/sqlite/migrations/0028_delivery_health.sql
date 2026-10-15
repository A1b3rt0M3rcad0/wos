-- Preserve confirmed successful delivery time across explicit redelivery.
ALTER TABLE integration_deliveries ADD COLUMN last_successful_at BIGINT;
UPDATE integration_deliveries SET last_successful_at=(
 SELECT finished_at FROM delivery_attempts a
 WHERE a.delivery_id=integration_deliveries.id AND a.fencing_token=integration_deliveries.fencing_token
) WHERE status='succeeded';
-- Scoped queue aggregation must avoid loading body_json/url/secret_ref.
CREATE INDEX delivery_scope_health ON integration_deliveries(namespace_id,outcome_id,status,next_attempt,id);
