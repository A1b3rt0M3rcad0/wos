WITH required_proof AS MATERIALIZED (
 SELECT c.id,c.criterion_revision FROM success_criteria c
 JOIN criterion_revisions r ON r.tenant_id=c.tenant_id AND r.outcome_id=c.outcome_id AND r.criterion_id=c.id AND r.criterion_revision=c.criterion_revision
 LEFT JOIN LATERAL (SELECT lifecycle,tenant_id,outcome_id FROM work_items w WHERE w.id=c.owner_id OFFSET 0) w ON w.tenant_id=c.tenant_id AND w.outcome_id=c.outcome_id AND c.owner_kind='work_item'
 LEFT JOIN LATERAL (SELECT lifecycle,tenant_id,outcome_id FROM objectives o WHERE o.id=c.owner_id OFFSET 0) o ON o.tenant_id=c.tenant_id AND o.outcome_id=c.outcome_id AND c.owner_kind='objective'
 WHERE c.tenant_id=? AND c.outcome_id=? AND c.status='active' AND r.required=1
 AND (c.owner_kind='outcome' OR (c.owner_kind='work_item' AND w.lifecycle!='cancelled') OR (c.owner_kind='objective' AND o.lifecycle!='cancelled'))
), current_proof AS MATERIALIZED (
 SELECT p.criterion_id,a.id,a.criterion_revision,a.result FROM criterion_current_assessments p
 JOIN LATERAL (SELECT id,criterion_revision,result FROM criterion_assessments a WHERE a.tenant_id=p.tenant_id AND a.outcome_id=p.outcome_id AND a.id=p.assessment_id OFFSET 0) a ON TRUE
 WHERE p.tenant_id=? AND p.outcome_id=?
), invalid_source AS MATERIALIZED (
 SELECT DISTINCT p.assessment_id FROM criterion_assessment_evidence p
 LEFT JOIN LATERAL (SELECT id,lifecycle,tenant_id,outcome_id FROM evidence e WHERE e.id=p.evidence_id OFFSET 0) e ON e.tenant_id=p.tenant_id AND e.outcome_id=p.outcome_id
 WHERE p.tenant_id=? AND p.outcome_id=? AND (e.id IS NULL OR e.lifecycle!='registered')
)
SELECT (SELECT COUNT(*) FROM required_proof),
COALESCE(SUM(CASE WHEN a.result='met' AND (a.criterion_id,a.criterion_revision) IN (SELECT id,criterion_revision FROM required_proof) AND a.id NOT IN (SELECT assessment_id FROM invalid_source) THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN a.result='waived' AND (a.criterion_id,a.criterion_revision) IN (SELECT id,criterion_revision FROM required_proof) THEN 1 ELSE 0 END),0)
FROM current_proof a
