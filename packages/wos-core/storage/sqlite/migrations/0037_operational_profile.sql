-- Operational profiles are opt-in so databases upgraded from older releases
-- retain their existing completion semantics.
ALTER TABLE outcomes ADD COLUMN operational_profile TEXT NOT NULL DEFAULT 'compatibility_v1'
    CHECK (operational_profile IN ('compatibility_v1', 'operational_v2'));

ALTER TABLE objectives ADD COLUMN completion_mode TEXT NOT NULL DEFAULT 'manual'
    CHECK (completion_mode IN ('manual', 'automatic'));

-- SQLite INTEGER is signed. Keep the unsigned cycle as canonical decimal text
-- so values through math.MaxUint64 survive without rounding or truncation.
ALTER TABLE objectives ADD COLUMN completion_cycle TEXT NOT NULL DEFAULT '1'
    CHECK (
        length(completion_cycle) BETWEEN 1 AND 20
        AND completion_cycle NOT GLOB '*[^0-9]*'
        AND substr(completion_cycle, 1, 1) <> '0'
        AND (length(completion_cycle) < 20 OR completion_cycle <= '18446744073709551615')
    );
