ALTER TABLE gitspace_configs
    ADD COLUMN gconf_is_marked_for_soft_reset BOOLEAN NOT NULL DEFAULT FALSE;