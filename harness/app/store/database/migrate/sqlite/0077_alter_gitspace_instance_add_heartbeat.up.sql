ALTER TABLE gitspaces ADD COLUMN gits_last_heartbeat BIGINT;
ALTER TABLE gitspaces DROP COLUMN gits_last_used;
ALTER TABLE gitspaces ADD COLUMN gits_last_used int64 NULL;