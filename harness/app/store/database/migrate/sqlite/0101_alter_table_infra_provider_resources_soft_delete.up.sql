ALTER TABLE infra_provider_resources
    RENAME COLUMN ipreso_opentofu_params TO ipreso_metadata;
ALTER TABLE infra_provider_resources
    ADD COLUMN ipreso_is_deleted BOOL NOT NULL DEFAULT false;
ALTER TABLE infra_provider_resources
    ADD COLUMN ipreso_deleted BIGINT;

DROP INDEX infra_provider_resources_uid_space_id;
CREATE UNIQUE INDEX infra_provider_resources_uid_space_id_config_id_created ON infra_provider_resources
    (ipreso_uid, ipreso_space_id, ipreso_infra_provider_config_id, ipreso_created);
