ALTER TABLE outcomes ADD COLUMN completion_policy TEXT NOT NULL DEFAULT 'compatible' CHECK (completion_policy IN ('compatible','strict_work'));
