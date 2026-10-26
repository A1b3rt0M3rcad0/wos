-- Immutable documentary notes, separate from contracted execution activity.
CREATE TABLE entity_notes (
 id TEXT PRIMARY KEY,
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 subject_kind TEXT NOT NULL,
 subject_id TEXT NOT NULL,
 criterion_id TEXT,
 principal_id TEXT NOT NULL,
 outcome_revision TEXT COLLATE BINARY NOT NULL CHECK (length(outcome_revision)=20 AND outcome_revision NOT GLOB '*[^0-9]*' AND outcome_revision>'00000000000000000000' AND outcome_revision<='18446744073709551615'),
 supersedes_note_id TEXT,
 payload_json TEXT NOT NULL,
 UNIQUE(namespace_id,outcome_id,id),
 UNIQUE(namespace_id,outcome_id,subject_kind,subject_id,id),
 FOREIGN KEY(namespace_id,outcome_id,subject_id,subject_kind) REFERENCES entity_refs(namespace_id,outcome_id,id,kind),
 FOREIGN KEY(namespace_id,outcome_id,criterion_id,subject_id) REFERENCES success_criteria(namespace_id,outcome_id,id,owner_id),
 FOREIGN KEY(namespace_id,outcome_id,subject_kind,subject_id,supersedes_note_id) REFERENCES entity_notes(namespace_id,outcome_id,subject_kind,subject_id,id)
);
CREATE INDEX entity_notes_by_subject ON entity_notes(namespace_id,outcome_id,subject_kind,subject_id,outcome_revision,id);
CREATE INDEX entity_notes_by_criterion ON entity_notes(namespace_id,outcome_id,subject_kind,subject_id,criterion_id,outcome_revision,id);
CREATE INDEX entity_notes_by_revision ON entity_notes(namespace_id,outcome_id,outcome_revision,id);
CREATE INDEX entity_notes_corrections ON entity_notes(namespace_id,outcome_id,supersedes_note_id);
CREATE TABLE entity_note_references (
 namespace_id TEXT NOT NULL,
 outcome_id TEXT NOT NULL,
 note_id TEXT NOT NULL,
 reference_kind TEXT NOT NULL CHECK(reference_kind IN ('artifact','evidence')),
 reference_id TEXT NOT NULL,
 PRIMARY KEY(namespace_id,outcome_id,note_id,reference_kind,reference_id),
 FOREIGN KEY(namespace_id,outcome_id,note_id) REFERENCES entity_notes(namespace_id,outcome_id,id),
 FOREIGN KEY(namespace_id,outcome_id,reference_id,reference_kind) REFERENCES entity_refs(namespace_id,outcome_id,id,kind)
);
-- Recent subject history uses the original immutable event order.
CREATE INDEX workspace_events_subject_order ON domain_events(namespace_id,outcome_id,aggregate_kind,aggregate_id,outcome_revision,event_index);
