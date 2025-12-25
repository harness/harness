ALTER TABLE gitspace_events RENAME COLUMN geven_event to geven_state;

ALTER TABLE gitspace_events ADD COLUMN geven_space_id INTEGER NOT NULL;

ALTER TABLE gitspaces DROP COLUMN gits_access_key;

ALTER TABLE gitspaces DROP COLUMN gits_access_type;

ALTER TABLE gitspaces DROP COLUMN gits_machine_user;