CREATE TABLE usergroups
(
    usergroup_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    usergroup_identifier  TEXT NOT NULL,
    usergroup_name        TEXT NOT NULL,
    usergroup_description TEXT,
    usergroup_space_id    INTEGER NOT NULL,
    usergroup_created     BIGINT,
    usergroup_updated     BIGINT,

    CONSTRAINT fk_usergroup_space_id FOREIGN KEY (usergroup_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX usergroups_space_id_identifier ON usergroups (usergroup_space_id, LOWER(usergroup_identifier));

