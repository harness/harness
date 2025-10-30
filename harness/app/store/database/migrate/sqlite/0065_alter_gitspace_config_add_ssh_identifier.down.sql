ALTER TABLE gitspace_configs DROP COLUMN gconf_ssh_token_identifier;
ALTER TABLE gitspaces DROP COLUMN gits_access_key_ref;
ALTER TABLE gitspaces ADD COLUMN gits_access_key TEXT;