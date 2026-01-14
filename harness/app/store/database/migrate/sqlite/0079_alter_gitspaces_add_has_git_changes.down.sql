ALTER TABLE gitspaces DROP COLUMN gits_has_git_changes;
ALTER TABLE gitspaces ADD COLUMN gits_tracked_changes TEXT;