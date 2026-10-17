CREATE TABLE delivery_namespace_schedule (
    namespace_id TEXT PRIMARY KEY REFERENCES namespaces(id),
    last_turn BIGINT NOT NULL DEFAULT 0 CHECK(last_turn >= 0)
);
INSERT INTO delivery_namespace_schedule(namespace_id) SELECT id FROM namespaces;
CREATE INDEX delivery_namespace_turn ON delivery_namespace_schedule(last_turn,namespace_id);
CREATE INDEX delivery_namespace_due ON integration_deliveries(namespace_id,status,next_attempt,id);
CREATE INDEX delivery_namespace_leased ON integration_deliveries(namespace_id,endpoint_id,lease_until) WHERE status='leased';
CREATE INDEX delivery_namespace_capacity ON integration_deliveries(namespace_id,lease_until) WHERE status='leased';
