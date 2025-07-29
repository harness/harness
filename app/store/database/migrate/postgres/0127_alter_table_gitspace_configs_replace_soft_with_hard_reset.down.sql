-- First drop the hard reset column
ALTER TABLE gitspace_configs DROP COLUMN gconf_is_marked_for_infra_reset;

-- Then re-add the soft reset column
ALTER TABLE gitspace_configs
    ADD COLUMN gconf_is_marked_for_soft_reset BOOLEAN NOT NULL DEFAULT FALSE;
