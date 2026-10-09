-- Public deployment trust and immutable operator recovery receipts only.
CREATE TABLE server_issuer_public_keys (
 key_id TEXT PRIMARY KEY, server_id TEXT NOT NULL, fingerprint TEXT NOT NULL UNIQUE, state_json TEXT NOT NULL
);
CREATE TABLE server_issuer_recoveries (
 replacement_key_id TEXT PRIMARY KEY REFERENCES server_issuer_public_keys(key_id),
 intent_digest TEXT NOT NULL, state_json TEXT NOT NULL
);
