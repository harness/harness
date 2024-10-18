ALTER TABLE gitspace_configs
    ADD COLUMN gconf_is_marked_for_deletion BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE gitspace_configs
SET gconf_is_marked_for_deletion = gconf_is_deleted;