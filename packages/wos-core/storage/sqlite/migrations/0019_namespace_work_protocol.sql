CREATE TABLE namespace_work_protocol (
 namespace_id TEXT PRIMARY KEY REFERENCES namespaces(id),
 version INTEGER NOT NULL CHECK(version>0),
 phase TEXT NOT NULL CHECK(phase IN ('legacy','draining','contracts_v1')),
 writer_epoch INTEGER NOT NULL CHECK(writer_epoch IN (0,1)),
 state_json TEXT NOT NULL
);
