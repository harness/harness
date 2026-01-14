ALTER TABLE gitspace_configs
    ADD COLUMN gconf_is_deleted BOOLEAN NOT NULL DEFAULT FALSE;