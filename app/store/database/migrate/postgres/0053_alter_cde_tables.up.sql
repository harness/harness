ALTER TABLE gitspace_events RENAME COLUMN geven_state to geven_event;

ALTER TABLE gitspace_events DROP COLUMN geven_space_id;

ALTER TABLE gitspaces ADD COLUMN gits_access_key TEXT;

ALTER TABLE gitspaces ADD COLUMN gits_access_type TEXT;

ALTER TABLE gitspaces ADD COLUMN gits_machine_user TEXT;