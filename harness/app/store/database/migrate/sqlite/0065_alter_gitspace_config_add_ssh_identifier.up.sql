ALTER TABLE gitspace_configs ADD COLUMN gconf_ssh_token_identifier TEXT DEFAULT '';
ALTER TABLE gitspaces ADD COLUMN gits_access_key_ref TEXT;
ALTER TABLE gitspaces DROP COLUMN gits_access_key;