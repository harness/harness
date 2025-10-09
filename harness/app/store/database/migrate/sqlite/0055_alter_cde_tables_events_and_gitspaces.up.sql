CREATE TABLE gitspace_events_temp
(
    geven_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    geven_event       TEXT    NOT NULL,
    geven_created     BIGINT  NOT NULL,
    geven_entity_type TEXT    NOT NULL,
    geven_entity_uid  TEXT,
    geven_entity_id   INTEGER NOT NULL
);

INSERT INTO gitspace_events_temp (geven_event, geven_created, geven_entity_type, geven_entity_id)
SELECT geven_event, geven_created, 'gitspaceConfig' AS geven_entity_type, geven_gitspace_config_id
FROM gitspace_events;

DROP TABLE gitspace_events;

ALTER TABLE gitspace_events_temp
    RENAME TO gitspace_events;

CREATE INDEX gitspace_events_entity_id ON gitspace_events (geven_entity_id);

CREATE TABLE gitspaces_temp
(
    gits_id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    gits_gitspace_config_id INTEGER NOT NULL,
    gits_url                TEXT,
    gits_state              TEXT    NOT NULL,
    gits_user_uid           TEXT    NOT NULL,
    gits_resource_usage     TEXT,
    gits_space_id           INTEGER NOT NULL,
    gits_created            BIGINT  NOT NULL,
    gits_updated            BIGINT  NOT NULL,
    gits_last_used          BIGINT  NOT NULL,
    gits_total_time_used    BIGINT  NOT NULL,
    gits_tracked_changes    TEXT,
    gits_access_key         TEXT,
    gits_access_type        TEXT,
    gits_machine_user       TEXT,
    gits_uid                TEXT    NOT NULL,
    CONSTRAINT fk_gits_gitspace_config_id FOREIGN KEY (gits_gitspace_config_id)
        REFERENCES gitspace_configs (gconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,
    CONSTRAINT fk_gits_space_id FOREIGN KEY (gits_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

INSERT INTO gitspaces_temp (gits_gitspace_config_id, gits_url, gits_state, gits_user_uid, gits_resource_usage,
                            gits_space_id, gits_created, gits_updated, gits_last_used, gits_total_time_used,
                            gits_tracked_changes, gits_access_key, gits_access_type,
                            gits_machine_user, gits_uid)
SELECT g.gits_gitspace_config_id,
       g.gits_url,
       g.gits_state,
       g.gits_user_uid,
       g.gits_resource_usage,
       g.gits_space_id,
       g.gits_created,
       g.gits_updated,
       g.gits_last_used,
       g.gits_total_time_used,
       g.gits_tracked_changes,
       g.gits_access_key,
       g.gits_access_type,
       g.gits_machine_user,
       gconf.gconf_uid AS gits_uid
FROM gitspaces g
         JOIN gitspace_configs gconf ON g.gits_gitspace_config_id = gconf.gconf_id;

DROP INDEX gitspaces_gitspace_config_id_space_id;
DROP TABLE gitspaces;

ALTER TABLE gitspaces_temp
    RENAME TO gitspaces;

CREATE UNIQUE INDEX gitspaces_uid_space_id ON gitspaces (gits_uid, gits_space_id);