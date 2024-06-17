CREATE TABLE gitspace_events_temp
(
    geven_id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    geven_gitspace_config_id INTEGER NOT NULL,
    geven_event              TEXT    NOT NULL,
    geven_created            BIGINT  NOT NULL,
    CONSTRAINT fk_geven_gitspace_config_id FOREIGN KEY (geven_gitspace_config_id)
        REFERENCES gitspace_configs (gconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);
INSERT INTO gitspace_events_temp (geven_gitspace_config_id, geven_event, geven_created)
SELECT geven_entity_id, geven_event, geven_created
FROM gitspace_events;
DROP INDEX gitspace_events_entity_id;
DROP TABLE gitspace_events;
ALTER TABLE gitspace_events_temp
    RENAME TO gitspace_events;



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

INSERT INTO gitspaces_temp (gits_gitspace_config_id, gits_url, gits_state, gits_user_uid, gits_resource_usage,
                            gits_space_id, gits_created, gits_updated, gits_last_used, gits_total_time_used,
                            gits_tracked_changes, gits_access_key, gits_access_type,
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
       gits_tracked_changes,
       gits_access_key,
       gits_access_type,
       gits_machine_user
FROM gitspaces;

DROP INDEX gitspaces_uid_space_id;
DROP TABLE gitspaces;

ALTER TABLE gitspaces_temp
    RENAME TO gitspaces;

CREATE UNIQUE INDEX gitspaces_gitspace_config_id_space_id ON gitspaces (gits_gitspace_config_id, gits_space_id);

