CREATE TABLE outcome_context_entries (
 namespace_id TEXT NOT NULL, outcome_id TEXT NOT NULL,context_key TEXT NOT NULL,canonical_value TEXT NOT NULL,
 PRIMARY KEY(namespace_id,outcome_id,context_key), FOREIGN KEY(namespace_id,outcome_id) REFERENCES outcomes(namespace_id,id)
);
CREATE INDEX outcome_context_lookup ON outcome_context_entries(namespace_id,context_key,canonical_value,outcome_id);
CREATE INDEX outcome_creator_lookup ON domain_events(namespace_id,event_type,principal_id,outcome_id);
CREATE INDEX outcome_discovery_lifecycle ON outcomes(namespace_id,lifecycle,created_at,id);
CREATE INDEX outcome_discovery_priority ON outcomes(namespace_id,priority,created_at,id);
CREATE UNIQUE INDEX external_reference_unique_address ON outcome_external_references(namespace_id,provider,context_kind,external_id);
