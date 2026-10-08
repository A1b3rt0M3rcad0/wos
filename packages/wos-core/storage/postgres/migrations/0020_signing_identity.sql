CREATE UNIQUE INDEX credentials_id_namespace ON credentials(id,namespace_id);
CREATE TABLE signing_keys (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), id TEXT NOT NULL,
 principal_id TEXT NOT NULL REFERENCES principals(id), purpose TEXT NOT NULL,
 fingerprint TEXT NOT NULL, status TEXT NOT NULL, version BIGINT NOT NULL,
 state_json TEXT NOT NULL, PRIMARY KEY(namespace_id,id), UNIQUE(namespace_id,fingerprint),
 CHECK(status IN ('active','retired','revoked')), CHECK(version>0)
);
CREATE INDEX signing_keys_principal ON signing_keys(namespace_id,principal_id,purpose,status,id);
CREATE TABLE signing_key_enrollments (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), id TEXT NOT NULL,
 principal_id TEXT NOT NULL REFERENCES principals(id), credential_id TEXT NOT NULL,
 version BIGINT NOT NULL, expires_at BIGINT NOT NULL, consumed BIGINT NOT NULL,
 state_json TEXT NOT NULL, PRIMARY KEY(namespace_id,id),
 FOREIGN KEY(credential_id,namespace_id) REFERENCES credentials(id,namespace_id), CHECK(version>0), CHECK(consumed IN (0,1))
);
CREATE TABLE credential_policies (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), credential_id TEXT NOT NULL,
 principal_id TEXT NOT NULL REFERENCES principals(id), version BIGINT NOT NULL,
 state_json TEXT NOT NULL, PRIMARY KEY(namespace_id,credential_id),
 FOREIGN KEY(credential_id,namespace_id) REFERENCES credentials(id,namespace_id), CHECK(version>0)
);
CREATE TABLE work_acceptance_policies (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), outcome_key TEXT NOT NULL, work_key TEXT NOT NULL,
 version BIGINT NOT NULL, state_json TEXT NOT NULL,
 PRIMARY KEY(namespace_id,outcome_key,work_key), CHECK(version>0)
);
CREATE TABLE principal_review_groups (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), principal_id TEXT NOT NULL REFERENCES principals(id),
 separation_group TEXT NOT NULL, PRIMARY KEY(namespace_id,principal_id)
);
CREATE TABLE signing_security_results (
 namespace_id TEXT NOT NULL REFERENCES namespaces(id), principal_id TEXT NOT NULL REFERENCES principals(id),
 idempotency_key TEXT NOT NULL, fingerprint TEXT NOT NULL, result_json TEXT NOT NULL,
 PRIMARY KEY(namespace_id,principal_id,idempotency_key)
);
