CREATE TABLE outcome_external_references (
 namespace_id TEXT NOT NULL,outcome_id TEXT NOT NULL,
 provider TEXT NOT NULL,context_kind TEXT NOT NULL,external_id TEXT NOT NULL,url TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(namespace_id,outcome_id,provider,context_kind,external_id),
 FOREIGN KEY(namespace_id,outcome_id) REFERENCES outcomes(namespace_id,id)
);
CREATE INDEX external_reference_discovery ON outcome_external_references(namespace_id,provider,context_kind,external_id,outcome_id);
