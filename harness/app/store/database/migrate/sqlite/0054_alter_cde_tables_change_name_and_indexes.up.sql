ALTER TABLE infra_provider_configs RENAME COLUMN ipconf_name TO ipconf_display_name;
ALTER TABLE infra_provider_resources RENAME COLUMN ipreso_name TO ipreso_display_name;
ALTER TABLE gitspace_configs RENAME COLUMN gconf_name TO gconf_display_name;

CREATE TABLE infra_provider_configs_temp
(
    ipconf_id           INTEGER PRIMARY KEY AUTOINCREMENT,
    ipconf_uid          TEXT    NOT NULL,
    ipconf_display_name TEXT    NOT NULL,
    ipconf_type         TEXT    NOT NULL,
    ipconf_space_id     INTEGER NOT NULL,
    ipconf_created      BIGINT  NOT NULL,
    ipconf_updated      BIGINT  NOT NULL,
    CONSTRAINT fk_ipconf_space_id FOREIGN KEY (ipconf_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE infra_provider_templates_temp
(
    iptemp_id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    iptemp_uid                      TEXT    NOT NULL,
    iptemp_infra_provider_config_id INTEGER NOT NULL,
    iptemp_description              TEXT    NOT NULL,
    iptemp_space_id                 INTEGER NOT NULL,
    iptemp_data                     BYTEA   NOT NULL,
    iptemp_created                  BIGINT  NOT NULL,
    iptemp_updated                  BIGINT  NOT NULL,
    iptemp_version                  INTEGER NOT NULL,
    CONSTRAINT fk_iptemp_space_id FOREIGN KEY (iptemp_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE,
    CONSTRAINT fk_infra_provider_config_id FOREIGN KEY (iptemp_infra_provider_config_id)
        REFERENCES infra_provider_configs (ipconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT
);

CREATE TABLE infra_provider_resources_temp
(
    ipreso_id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    ipreso_uid                        TEXT    NOT NULL,
    ipreso_display_name               TEXT    NOT NULL,
    ipreso_infra_provider_config_id   INTEGER NOT NULL,
    ipreso_type                       TEXT    NOT NULL,
    ipreso_space_id                   INTEGER NOT NULL,
    ipreso_created                    BIGINT  NOT NULL,
    ipreso_updated                    BIGINT  NOT NULL,
    ipreso_cpu                        TEXT    NOT NULL,
    ipreso_memory                     TEXT    NOT NULL,
    ipreso_disk                       TEXT    NOT NULL,
    ipreso_network                    TEXT,
    ipreso_region                     TEXT    NOT NULL,
    ipreso_opentofu_params            JSONB,
    ipreso_gateway_host               TEXT,
    ipreso_gateway_port               TEXT,
    ipreso_infra_provider_template_id INTEGER,
    CONSTRAINT fk_ipreso_infra_provider_template_id FOREIGN KEY (ipreso_infra_provider_template_id)
        REFERENCES infra_provider_templates (iptemp_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,
    CONSTRAINT fk_ipreso_infra_provider_config_id FOREIGN KEY (ipreso_infra_provider_config_id)
        REFERENCES infra_provider_configs (ipconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,
    CONSTRAINT fk_ipreso_space_id FOREIGN KEY (ipreso_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE gitspace_configs_temp
(
    gconf_id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    gconf_uid                        TEXT    NOT NULL,
    gconf_display_name               TEXT    NOT NULL,
    gconf_ide                        TEXT    NOT NULL,
    gconf_infra_provider_resource_id INTEGER NOT NULL,
    gconf_code_auth_type             TEXT    NOT NULL,
    gconf_code_auth_id               TEXT    NOT NULL,
    gconf_code_repo_type             TEXT    NOT NULL,
    gconf_code_repo_is_private       BOOLEAN NOT NULL,
    gconf_code_repo_url              TEXT    NOT NULL,
    gconf_devcontainer_path          TEXT,
    gconf_branch                     TEXT,
    gconf_user_uid                   TEXT    NOT NULL,
    gconf_space_id                   INTEGER NOT NULL,
    gconf_created                    BIGINT  NOT NULL,
    gconf_updated                    BIGINT  NOT NULL,
    CONSTRAINT fk_gconf_infra_provider_resource_id FOREIGN KEY (gconf_infra_provider_resource_id)
        REFERENCES infra_provider_resources (ipreso_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,
    CONSTRAINT fk_gconf_space_id FOREIGN KEY (gconf_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE gitspaces_temp
(
    gits_id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    gits_gitspace_config_id   INTEGER NOT NULL,
    gits_url                  TEXT,
    gits_state                TEXT    NOT NULL,
    gits_user_uid             TEXT    NOT NULL,
    gits_resource_usage       TEXT,
    gits_space_id             INTEGER NOT NULL,
    gits_created              BIGINT  NOT NULL,
    gits_updated              BIGINT  NOT NULL,
    gits_last_used            BIGINT  NOT NULL,
    gits_total_time_used      BIGINT  NOT NULL,
    gits_infra_provisioned_id INTEGER,
    gits_tracked_changes      TEXT,
    gits_access_key           TEXT,
    gits_access_type          TEXT,
    gits_machine_user         TEXT,
    CONSTRAINT fk_gits_gitspace_config_id FOREIGN KEY (gits_gitspace_config_id)
        REFERENCES gitspace_configs (gconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,
    CONSTRAINT fk_gits_infra_provisioned_id FOREIGN KEY (gits_infra_provisioned_id)
        REFERENCES infra_provisioned (iprov_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT fk_gits_space_id FOREIGN KEY (gits_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

INSERT INTO infra_provider_configs_temp (ipconf_uid, ipconf_display_name, ipconf_type, ipconf_space_id, ipconf_created,
                                         ipconf_updated)
SELECT ipconf_uid, ipconf_display_name, ipconf_type, ipconf_space_id, ipconf_created, ipconf_updated
FROM infra_provider_configs;

INSERT INTO infra_provider_templates_temp (iptemp_uid, iptemp_infra_provider_config_id, iptemp_description,
                                           iptemp_space_id, iptemp_data, iptemp_created, iptemp_updated, iptemp_version)
SELECT iptemp_uid,
       iptemp_infra_provider_config_id,
       iptemp_description,
       iptemp_space_id,
       iptemp_data,
       iptemp_created,
       iptemp_updated,
       iptemp_version
FROM infra_provider_templates;

INSERT INTO infra_provider_resources_temp (ipreso_uid, ipreso_display_name, ipreso_infra_provider_config_id,
                                           ipreso_type, ipreso_space_id, ipreso_created, ipreso_updated, ipreso_cpu,
                                           ipreso_memory, ipreso_disk, ipreso_network, ipreso_region,
                                           ipreso_opentofu_params, ipreso_gateway_host, ipreso_gateway_port,
                                           ipreso_infra_provider_template_id)
SELECT ipreso_uid,
       ipreso_display_name,
       ipreso_infra_provider_config_id,
       ipreso_type,
       ipreso_space_id,
       ipreso_created,
       ipreso_updated,
       ipreso_cpu,
       ipreso_memory,
       ipreso_disk,
       ipreso_network,
       ipreso_region,
       ipreso_opentofu_params,
       ipreso_gateway_host,
       ipreso_gateway_port,
       ipreso_infra_provider_template_id
FROM infra_provider_resources;

INSERT INTO gitspace_configs_temp (gconf_uid, gconf_display_name, gconf_ide, gconf_infra_provider_resource_id,
                                   gconf_code_auth_type, gconf_code_auth_id, gconf_code_repo_type,
                                   gconf_code_repo_is_private, gconf_code_repo_url, gconf_devcontainer_path,
                                   gconf_branch, gconf_user_uid, gconf_space_id, gconf_created, gconf_updated)
SELECT gconf_uid,
       gconf_display_name,
       gconf_ide,
       gconf_infra_provider_resource_id,
       gconf_code_auth_type,
       gconf_code_auth_id,
       gconf_code_repo_type,
       gconf_code_repo_is_private,
       gconf_code_repo_url,
       gconf_devcontainer_path,
       gconf_branch,
       gconf_user_uid,
       gconf_space_id,
       gconf_created,
       gconf_updated
FROM gitspace_configs;

INSERT INTO gitspaces_temp (gits_gitspace_config_id, gits_url, gits_state, gits_user_uid, gits_resource_usage,
                            gits_space_id, gits_created, gits_updated, gits_last_used, gits_total_time_used,
                            gits_infra_provisioned_id, gits_tracked_changes, gits_access_key, gits_access_type,
                            gits_machine_user)
SELECT gits_gitspace_config_id,
       gits_url,
       gits_state,
       gits_user_uid,
       gits_resource_usage,
       gits_space_id,
       gits_created,
       gits_updated,
       gits_last_used,
       gits_total_time_used,
       gits_infra_provisioned_id,
       gits_tracked_changes,
       gits_access_key,
       gits_access_type,
       gits_machine_user
FROM gitspaces;

DROP TABLE infra_provider_configs;
DROP TABLE infra_provider_templates;
DROP TABLE infra_provider_resources;
DROP TABLE gitspace_configs;
DROP TABLE gitspaces;

ALTER TABLE infra_provider_configs_temp
    RENAME TO infra_provider_configs;
ALTER TABLE infra_provider_templates_temp
    RENAME TO infra_provider_templates;
ALTER TABLE infra_provider_resources_temp
    RENAME TO infra_provider_resources;
ALTER TABLE gitspace_configs_temp
    RENAME TO gitspace_configs;
ALTER TABLE gitspaces_temp
    RENAME TO gitspaces;

CREATE UNIQUE INDEX infra_provider_configs_uid_space_id ON infra_provider_configs (ipconf_uid, ipconf_space_id);
CREATE UNIQUE INDEX infra_provider_templates_uid_space_id ON infra_provider_templates (iptemp_uid, iptemp_space_id);
CREATE UNIQUE INDEX infra_provider_resources_uid_space_id ON infra_provider_resources (ipreso_uid, ipreso_space_id);
CREATE UNIQUE INDEX gitspace_configs_uid_space_id ON gitspace_configs (gconf_uid, gconf_space_id);
CREATE UNIQUE INDEX gitspaces_gitspace_config_id_space_id ON gitspaces (gits_gitspace_config_id, gits_space_id);
