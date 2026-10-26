-- Explicit, versioned authorization history for Objective completion policy.
-- Existing completion_mode='automatic' rows are intentionally not backfilled:
-- they remain inactive until an authorized command records an exact-cycle
-- credential-bound authority.
CREATE TABLE objective_completion_authority_history (
    id TEXT PRIMARY KEY,
    namespace_id TEXT NOT NULL,
    outcome_id TEXT NOT NULL,
    objective_id TEXT NOT NULL,
    policy_version TEXT NOT NULL CHECK (
        length(policy_version) BETWEEN 1 AND 20
        AND policy_version NOT GLOB '*[^0-9]*'
        AND substr(policy_version, 1, 1) <> '0'
        AND (length(policy_version) < 20 OR policy_version <= '18446744073709551615')
    ),
    mode TEXT NOT NULL CHECK (mode IN ('manual', 'automatic')),
    completion_cycle TEXT NOT NULL CHECK (
        length(completion_cycle) BETWEEN 1 AND 20
        AND completion_cycle NOT GLOB '*[^0-9]*'
        AND substr(completion_cycle, 1, 1) <> '0'
        AND (length(completion_cycle) < 20 OR completion_cycle <= '18446744073709551615')
    ),
    delegator_principal_id TEXT NOT NULL,
    delegator_actor_json TEXT NOT NULL CHECK (json_valid(delegator_actor_json)),
    delegator_credential_id TEXT,
    credential_policy_revision TEXT NOT NULL DEFAULT '0' CHECK (
        length(credential_policy_revision) BETWEEN 1 AND 20
        AND credential_policy_revision NOT GLOB '*[^0-9]*'
        AND (credential_policy_revision = '0' OR substr(credential_policy_revision, 1, 1) <> '0')
        AND (length(credential_policy_revision) < 20 OR credential_policy_revision <= '18446744073709551615')
    ),
    operational_profile TEXT NOT NULL CHECK (operational_profile IN ('compatibility_v1', 'operational_v2')),
    completion_policy TEXT NOT NULL CHECK (completion_policy IN ('compatible', 'strict_work')),
    automation_actor_json TEXT NOT NULL CHECK (
        json_valid(automation_actor_json)
        AND automation_actor_json = '{"kind":"automation","provider":"wos","id":"objective_completion"}'
    ),
    reason TEXT NOT NULL CHECK (length(trim(reason)) BETWEEN 1 AND 4096),
    authorized_at BIGINT NOT NULL CHECK (authorized_at > 0),
    authority_json TEXT NOT NULL CHECK (json_valid(authority_json)),
    UNIQUE (namespace_id, outcome_id, objective_id, policy_version),
    UNIQUE (namespace_id, outcome_id, objective_id, id),
    FOREIGN KEY (namespace_id, outcome_id, objective_id)
        REFERENCES objectives(namespace_id, outcome_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (delegator_credential_id, namespace_id)
        REFERENCES credentials(id, namespace_id) ON DELETE RESTRICT,
    CHECK (
        (delegator_credential_id IS NULL AND credential_policy_revision = '0')
        OR (delegator_credential_id IS NOT NULL AND credential_policy_revision <> '0')
    ),
    CHECK (
        mode <> 'automatic'
        OR (delegator_credential_id IS NOT NULL AND credential_policy_revision <> '0'
            AND operational_profile = 'operational_v2' AND completion_policy = 'strict_work')
    )
);

CREATE INDEX objective_completion_authority_latest
    ON objective_completion_authority_history(namespace_id, outcome_id, objective_id, policy_version);

CREATE TRIGGER objective_completion_authority_immutable_update
BEFORE UPDATE ON objective_completion_authority_history
BEGIN SELECT RAISE(ABORT, 'Objective completion authority history is immutable'); END;

CREATE TRIGGER objective_completion_authority_immutable_delete
BEFORE DELETE ON objective_completion_authority_history
BEGIN SELECT RAISE(ABORT, 'Objective completion authority history is immutable'); END;
