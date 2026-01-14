ALTER TABLE infra_provider_configs
    ADD COLUMN ipconf_is_deleted BOOL NOT NULL DEFAULT false;
ALTER TABLE infra_provider_configs
    ADD COLUMN ipconf_deleted BIGINT;

DROP INDEX infra_provider_configs_uid_space_id;
CREATE UNIQUE INDEX infra_provider_configs_uid_space_id_created ON infra_provider_configs
    (ipconf_uid, ipconf_space_id, ipconf_created);
