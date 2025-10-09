CREATE TABLE IF NOT EXISTS cde_gateways
(
    cgate_id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    cgate_name                     TEXT   NOT NULL,
    cgate_group_name               TEXT   NOT NULL,
    cgate_region                   TEXT   NOT NULL,
    cgate_zone                     TEXT   NOT NULL,
    cgate_version                  TEXT   NOT NULL,
    cgate_health                   TEXT   NOT NULL,
    cgate_space_id                 BIGINT NOT NULL,
    cgate_infra_provider_config_id BIGINT NOT NULL,
    cgate_envoy_health             TEXT   NOT NULL,
    cgate_created                  BIGINT NOT NULL,
    cgate_updated                  BIGINT NOT NULL,
    CONSTRAINT fk_cgate_space_id FOREIGN KEY (cgate_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT fk_cgate_infra_provider_config_id FOREIGN KEY (cgate_infra_provider_config_id)
        REFERENCES infra_provider_configs (ipconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

CREATE UNIQUE INDEX IF NOT EXISTS cde_gateways_space_id_infra_config_id_region_group_name_name
    ON cde_gateways (cgate_space_id, cgate_infra_provider_config_id, cgate_region, cgate_group_name, cgate_name);