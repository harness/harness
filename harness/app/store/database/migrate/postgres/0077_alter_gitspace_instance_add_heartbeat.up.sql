ALTER TABLE gitspaces ADD COLUMN gits_last_heartbeat BIGINT;
ALTER TABLE gitspaces ALTER COLUMN gits_last_used DROP NOT NULL;
