DROP INDEX infra_provider_configs_uid_space_id_created;

ALTER TABLE infra_provider_configs
    DROP COLUMN ipconf_is_deleted;
ALTER TABLE infra_provider_configs
    DROP COLUMN ipconf_deleted;


CREATE UNIQUE INDEX infra_provider_configs_uid_space_id ON infra_provider_configs (ipconf_uid, ipconf_space_id);
