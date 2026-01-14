DROP INDEX infra_provider_resources_uid_space_id_config_id_created;

ALTER TABLE infra_provider_resources
    RENAME COLUMN ipreso_metadata TO ipreso_opentofu_params;
ALTER TABLE infra_provider_resources
    DROP COLUMN ipreso_is_deleted;
ALTER TABLE infra_provider_resources
    DROP COLUMN ipreso_deleted;


CREATE UNIQUE INDEX infra_provider_resources_uid_space_id ON infra_provider_resources (ipreso_uid, ipreso_space_id);
