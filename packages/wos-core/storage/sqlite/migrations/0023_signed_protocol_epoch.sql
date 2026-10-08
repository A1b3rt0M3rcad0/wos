-- Preserve every existing row/JSON byte while expanding the original metadata
-- constraints. There are no inbound FKs to this coordination metadata table.
-- Old 0.2 writers still read this same table and reject epoch 2/unknown phases.
ALTER TABLE namespace_work_protocol RENAME TO namespace_work_protocol_before_signed;
CREATE TABLE namespace_work_protocol (
 namespace_id TEXT PRIMARY KEY REFERENCES namespaces(id),
 version INTEGER NOT NULL CHECK(version>0),
 phase TEXT NOT NULL CHECK(phase IN ('legacy','draining','contracts_v1','draining_to_signed_v2','signed_contracts_v2')),
 writer_epoch INTEGER NOT NULL CHECK(writer_epoch IN (0,1,2)),
 state_json TEXT NOT NULL
);
INSERT INTO namespace_work_protocol(namespace_id,version,phase,writer_epoch,state_json)
 SELECT namespace_id,version,phase,writer_epoch,state_json FROM namespace_work_protocol_before_signed;
DROP TABLE namespace_work_protocol_before_signed;
