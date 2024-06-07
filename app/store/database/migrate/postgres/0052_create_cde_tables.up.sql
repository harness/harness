CREATE TABLE infra_provider_configs
(
    ipconf_id       SERIAL PRIMARY KEY,
    ipconf_uid      TEXT    NOT NULL,
    ipconf_name     TEXT    NOT NULL,
    ipconf_type     TEXT    NOT NULL,
    ipconf_space_id INTEGER NOT NULL,
    ipconf_created  BIGINT  NOT NULL,
    ipconf_updated  BIGINT  NOT NULL,
    UNIQUE (ipconf_uid, ipconf_space_id),
    CONSTRAINT fk_ipconf_space_id FOREIGN KEY (ipconf_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE infra_provider_templates
(
    iptemp_id                       SERIAL PRIMARY KEY,
    iptemp_uid                      TEXT    NOT NULL,
    iptemp_infra_provider_config_id INTEGER NOT NULL,
    iptemp_description              TEXT    NOT NULL,
    iptemp_space_id                 INTEGER NOT NULL,
    iptemp_data                     BYTEA   NOT NULL,
    iptemp_created                  BIGINT  NOT NULL,
    iptemp_updated                  BIGINT  NOT NULL,
    iptemp_version                  INTEGER NOT NULL,
    UNIQUE (iptemp_uid, iptemp_space_id),
    CONSTRAINT fk_iptemp_space_id FOREIGN KEY (iptemp_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE,
    CONSTRAINT fk_infra_provider_config_id FOREIGN KEY (iptemp_infra_provider_config_id)
        REFERENCES infra_provider_configs (ipconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT
);

CREATE TABLE infra_provider_resources
(
    ipreso_id                         SERIAL PRIMARY KEY,
    ipreso_uid                        TEXT    NOT NULL,
    ipreso_name                       TEXT    NOT NULL,
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
    UNIQUE (ipreso_uid, ipreso_space_id),
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

CREATE TABLE gitspace_configs
(
    gconf_id                         SERIAL PRIMARY KEY,
    gconf_uid                        TEXT    NOT NULL,
    gconf_name                       TEXT    NOT NULL,
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
    UNIQUE (gconf_uid, gconf_space_id),
    CONSTRAINT fk_gconf_infra_provider_resource_id FOREIGN KEY (gconf_infra_provider_resource_id)
        REFERENCES infra_provider_resources (ipreso_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,
    CONSTRAINT fk_gconf_space_id FOREIGN KEY (gconf_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE infra_provisioned
(
    iprov_id                         SERIAL PRIMARY KEY,
    iprov_gitspace_id                INTEGER NOT NULL,
    iprov_type                       TEXT,
    iprov_infra_provider_resource_id INTEGER NOT NULL,
    iprov_space_id                   INTEGER NOT NULL,
    iprov_created                    BIGINT  NOT NULL,
    iprov_updated                    BIGINT  NOT NULL,
    iprov_response_metadata          BYTEA,
    iprov_opentofu_params            JSONB   NOT NULL,
    iprov_opentofu_template          BYTEA   NOT NULL,
    iprov_infra_status               TEXT    NOT NULL,
    iprov_server_host_ip             TEXT,
    iprov_server_host_port           TEXT,
    CONSTRAINT fk_iprov_infra_provider_resource_id FOREIGN KEY (iprov_infra_provider_resource_id)
        REFERENCES infra_provider_resources (ipreso_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT fk_iprov_space_id FOREIGN KEY (iprov_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION

);

CREATE TABLE gitspaces
(
    gits_id                   SERIAL PRIMARY KEY,
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
    UNIQUE (gits_gitspace_config_id, gits_space_id),
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

CREATE TABLE gitspace_events
(
    geven_id                 SERIAL PRIMARY KEY,
    geven_gitspace_config_id INTEGER NOT NULL,
    geven_state              TEXT    NOT NULL,
    geven_created            BIGINT  NOT NULL,
    geven_space_id           INTEGER NOT NULL,
    CONSTRAINT fk_geven_gitspace_config_id FOREIGN KEY (geven_gitspace_config_id)
        REFERENCES gitspace_configs (gconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE,
    CONSTRAINT fk_geven_space_id FOREIGN KEY (geven_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

ALTER TABLE infra_provisioned
    ADD CONSTRAINT fk_iprov_gitspace_id FOREIGN KEY (iprov_gitspace_id)
        REFERENCES gitspaces (gits_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION;
