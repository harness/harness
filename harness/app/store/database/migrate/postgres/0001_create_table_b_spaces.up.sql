CREATE TABLE spaces (
 space_id           SERIAL PRIMARY KEY
,space_version      INTEGER NOT NULL DEFAULT 0
,space_parent_id    INTEGER DEFAULT NULL
,space_uid          TEXT NOT NULL
,space_description  TEXT
,space_is_public    BOOLEAN NOT NULL
,space_created_by   INTEGER NOT NULL
,space_created      BIGINT NOT NULL
,space_updated      BIGINT NOT NULL

,CONSTRAINT fk_space_parent_id FOREIGN KEY (space_parent_id)
    REFERENCES spaces (space_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);