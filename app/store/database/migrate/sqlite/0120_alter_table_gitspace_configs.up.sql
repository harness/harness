ALTER TABLE gitspace_configs
    ADD COLUMN gconf_is_marked_for_reset BOOLEAN NOT NULL DEFAULT FALSE;