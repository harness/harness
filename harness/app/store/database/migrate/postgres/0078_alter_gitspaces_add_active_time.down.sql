ALTER TABLE gitspaces DROP COLUMN gits_active_time_started;
ALTER TABLE gitspaces DROP COLUMN gits_active_time_ended;
UPDATE gitspaces SET gits_total_time_used = 0;