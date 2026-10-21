-- Immutable unsigned delivery and reviewer-decision history. The current
-- delivery navigation is stored separately from WorkItem's operational row so
-- scoped foreign keys can protect every pointer without rewriting old rows.
CREATE UNIQUE INDEX work_contract_submissions_delivery_ref
    ON work_contract_submissions(namespace_id,outcome_id,work_item_id,contract_id,id,submission_digest);

CREATE TABLE task_deliveries (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    work_item_id TEXT NOT NULL,
    delivery_round TEXT NOT NULL CHECK (
        length(delivery_round) BETWEEN 1 AND 20
        AND delivery_round !~ '[^0-9]'
        AND delivery_round <> '0'
        AND substr(delivery_round,1,1) <> '0'
        AND (length(delivery_round) < 20 OR delivery_round <= '18446744073709551615')
    ),
    authority_kind TEXT NOT NULL CHECK (authority_kind IN ('legacy_lease','contracts_v1')),
    contract_id TEXT,
    submission_id TEXT,
    submission_digest TEXT,
    delivery_digest TEXT NOT NULL,
    definition_digest TEXT NOT NULL,
    payload_json TEXT NOT NULL CHECK (jsonb_typeof(payload_json::jsonb) IS NOT NULL),
    UNIQUE(namespace_id,outcome_id,id),
    UNIQUE(namespace_id,outcome_id,work_item_id,id),
    UNIQUE(namespace_id,outcome_id,work_item_id,id,delivery_round,delivery_digest),
    UNIQUE(namespace_id,outcome_id,work_item_id,delivery_round),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id)
        REFERENCES work_items(namespace_id,outcome_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,contract_id)
        REFERENCES work_contracts(namespace_id,outcome_id,work_item_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,contract_id,submission_id,submission_digest)
        REFERENCES work_contract_submissions(namespace_id,outcome_id,work_item_id,contract_id,id,submission_digest),
    CHECK (
        (authority_kind='legacy_lease' AND contract_id IS NULL AND submission_id IS NULL AND submission_digest IS NULL)
        OR
        (authority_kind='contracts_v1' AND contract_id IS NOT NULL AND submission_id IS NOT NULL AND submission_digest IS NOT NULL)
    )
);
CREATE INDEX task_deliveries_work_history
    ON task_deliveries(namespace_id,outcome_id,work_item_id,id);

CREATE TABLE task_delivery_decisions (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    work_item_id TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    delivery_round TEXT NOT NULL,
    delivery_digest TEXT NOT NULL,
    result TEXT NOT NULL CHECK (result IN ('accepted','changes_requested')),
    payload_json TEXT NOT NULL CHECK (jsonb_typeof(payload_json::jsonb) IS NOT NULL),
    UNIQUE(namespace_id,outcome_id,id),
    UNIQUE(namespace_id,outcome_id,work_item_id,delivery_id),
    UNIQUE(namespace_id,outcome_id,work_item_id,delivery_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id,delivery_round,delivery_digest)
);
CREATE INDEX task_delivery_decisions_history
    ON task_delivery_decisions(namespace_id,outcome_id,delivery_id,id);

-- Typed references are stored beside the canonical immutable payload so the
-- database can enforce that every referenced Artifact, Evidence record and
-- criterion revision belongs to this exact Task and Outcome.
CREATE TABLE task_delivery_criteria (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    work_item_id TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    snapshot_index INTEGER NOT NULL CHECK (snapshot_index >= 0),
    criterion_id TEXT NOT NULL,
    criterion_revision BIGINT NOT NULL CHECK (criterion_revision >= 1),
    required INTEGER NOT NULL CHECK (required IN (0,1)),
    verification_mode TEXT NOT NULL CHECK (verification_mode IN ('attestation','evidence_review','external_evaluation')),
    PRIMARY KEY(namespace_id,outcome_id,delivery_id,criterion_id),
    UNIQUE(namespace_id,outcome_id,delivery_id,snapshot_index),
    UNIQUE(namespace_id,outcome_id,delivery_id,criterion_id,criterion_revision),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,delivery_id)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id),
    FOREIGN KEY(namespace_id,outcome_id,criterion_id,criterion_revision)
        REFERENCES criterion_revisions(namespace_id,outcome_id,criterion_id,criterion_revision),
    FOREIGN KEY(namespace_id,outcome_id,criterion_id,work_item_id)
        REFERENCES success_criteria(namespace_id,outcome_id,id,owner_id)
);

CREATE TABLE task_delivery_artifacts (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    artifact_index INTEGER NOT NULL CHECK (artifact_index >= 0),
    artifact_id TEXT NOT NULL,
    source_version TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL DEFAULT '',
    PRIMARY KEY(namespace_id,outcome_id,delivery_id,artifact_index),
    UNIQUE(namespace_id,outcome_id,delivery_id,artifact_id),
    FOREIGN KEY(namespace_id,outcome_id,delivery_id)
        REFERENCES task_deliveries(namespace_id,outcome_id,id),
    FOREIGN KEY(namespace_id,outcome_id,artifact_id)
        REFERENCES artifacts(namespace_id,outcome_id,id)
);

CREATE TABLE task_delivery_evidence (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    evidence_index INTEGER NOT NULL CHECK (evidence_index >= 0),
    evidence_id TEXT NOT NULL,
    PRIMARY KEY(namespace_id,outcome_id,delivery_id,evidence_index),
    UNIQUE(namespace_id,outcome_id,delivery_id,evidence_id),
    FOREIGN KEY(namespace_id,outcome_id,delivery_id)
        REFERENCES task_deliveries(namespace_id,outcome_id,id),
    FOREIGN KEY(namespace_id,outcome_id,evidence_id)
        REFERENCES evidence(namespace_id,outcome_id,id)
);

CREATE TABLE task_delivery_criterion_evidence (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    criterion_id TEXT NOT NULL,
    criterion_revision BIGINT NOT NULL,
    evidence_index INTEGER NOT NULL CHECK (evidence_index >= 0),
    evidence_id TEXT NOT NULL,
    PRIMARY KEY(namespace_id,outcome_id,delivery_id,criterion_id,evidence_index),
    UNIQUE(namespace_id,outcome_id,delivery_id,criterion_id,evidence_id),
    FOREIGN KEY(namespace_id,outcome_id,delivery_id,criterion_id,criterion_revision)
        REFERENCES task_delivery_criteria(namespace_id,outcome_id,delivery_id,criterion_id,criterion_revision),
    FOREIGN KEY(namespace_id,outcome_id,delivery_id,evidence_id)
        REFERENCES task_delivery_evidence(namespace_id,outcome_id,delivery_id,evidence_id)
);
CREATE INDEX task_delivery_artifacts_by_artifact
    ON task_delivery_artifacts(namespace_id,outcome_id,artifact_id,delivery_id);
CREATE INDEX task_delivery_evidence_by_evidence
    ON task_delivery_evidence(namespace_id,outcome_id,evidence_id,delivery_id);

CREATE TABLE task_delivery_work_state (
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    work_item_id TEXT NOT NULL,
    pending_delivery_id TEXT,
    latest_delivery_id TEXT,
    latest_decision_id TEXT,
    delivery_round TEXT NOT NULL DEFAULT '0' CHECK (
        length(delivery_round) BETWEEN 1 AND 20
        AND delivery_round !~ '[^0-9]'
        AND (delivery_round='0' OR substr(delivery_round,1,1)<>'0')
        AND (length(delivery_round)<20 OR delivery_round<='18446744073709551615')
    ),
    PRIMARY KEY(namespace_id,outcome_id,work_item_id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id)
        REFERENCES work_items(namespace_id,outcome_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,pending_delivery_id)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,latest_delivery_id)
        REFERENCES task_deliveries(namespace_id,outcome_id,work_item_id,id),
    FOREIGN KEY(namespace_id,outcome_id,work_item_id,latest_delivery_id,latest_decision_id)
        REFERENCES task_delivery_decisions(namespace_id,outcome_id,work_item_id,delivery_id,id),
    CHECK (
        (delivery_round='0' AND pending_delivery_id IS NULL AND latest_delivery_id IS NULL AND latest_decision_id IS NULL)
        OR
        (delivery_round<>'0' AND latest_delivery_id IS NOT NULL AND
            ((pending_delivery_id IS NOT NULL AND pending_delivery_id=latest_delivery_id AND latest_decision_id IS NULL)
             OR (pending_delivery_id IS NULL AND latest_decision_id IS NOT NULL)))
    )
);
CREATE UNIQUE INDEX task_delivery_one_pending_per_work_item
    ON task_delivery_work_state(namespace_id,outcome_id,work_item_id)
    WHERE pending_delivery_id IS NOT NULL;
CREATE INDEX task_delivery_pending_queue
    ON task_delivery_work_state(namespace_id,outcome_id,work_item_id,latest_delivery_id)
    WHERE pending_delivery_id IS NOT NULL;

-- Immutable facts and their typed reference rows cannot be rewritten or
-- deleted. Work-state navigation remains mutable through versioned UoW saves.
CREATE FUNCTION reject_immutable_task_delivery_fact() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'TaskDelivery facts are immutable';
END;
$$;
CREATE TRIGGER task_deliveries_immutable BEFORE UPDATE OR DELETE ON task_deliveries FOR EACH ROW EXECUTE FUNCTION reject_immutable_task_delivery_fact();
CREATE TRIGGER task_delivery_decisions_immutable BEFORE UPDATE OR DELETE ON task_delivery_decisions FOR EACH ROW EXECUTE FUNCTION reject_immutable_task_delivery_fact();
CREATE TRIGGER task_delivery_criteria_immutable BEFORE UPDATE OR DELETE ON task_delivery_criteria FOR EACH ROW EXECUTE FUNCTION reject_immutable_task_delivery_fact();
CREATE TRIGGER task_delivery_artifacts_immutable BEFORE UPDATE OR DELETE ON task_delivery_artifacts FOR EACH ROW EXECUTE FUNCTION reject_immutable_task_delivery_fact();
CREATE TRIGGER task_delivery_evidence_immutable BEFORE UPDATE OR DELETE ON task_delivery_evidence FOR EACH ROW EXECUTE FUNCTION reject_immutable_task_delivery_fact();
CREATE TRIGGER task_delivery_criterion_evidence_immutable BEFORE UPDATE OR DELETE ON task_delivery_criterion_evidence FOR EACH ROW EXECUTE FUNCTION reject_immutable_task_delivery_fact();
CREATE FUNCTION guard_task_delivery_pending_pointer() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.pending_delivery_id IS NOT NULL AND NEW.pending_delivery_id IS NOT NULL AND OLD.pending_delivery_id IS DISTINCT FROM NEW.pending_delivery_id THEN
        RAISE EXCEPTION 'Task already has a pending TaskDelivery';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER task_delivery_work_state_single_pending BEFORE UPDATE OF pending_delivery_id ON task_delivery_work_state FOR EACH ROW EXECUTE FUNCTION guard_task_delivery_pending_pointer();
