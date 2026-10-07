ALTER TABLE credentials ADD COLUMN parent_digest TEXT;
CREATE INDEX credentials_parent ON credentials(parent_digest);
