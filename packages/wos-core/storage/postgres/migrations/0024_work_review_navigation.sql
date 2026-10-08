-- Add only navigation: immutable review targets/decisions remain in signed facts/cases.
ALTER TABLE work_items ADD COLUMN pending_review_case_id TEXT;
ALTER TABLE work_items ADD COLUMN latest_review_case_id TEXT;
ALTER TABLE work_items ADD COLUMN correction_review_case_id TEXT;
CREATE INDEX work_items_pending_review ON work_items(namespace_id,outcome_id,pending_review_case_id) WHERE pending_review_case_id IS NOT NULL;
CREATE INDEX domain_events_work_participants ON domain_events(namespace_id,outcome_id,aggregate_id,event_type,principal_id);
