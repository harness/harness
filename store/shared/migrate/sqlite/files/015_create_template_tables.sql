-- name: create-table-template

CREATE TABLE IF NOT EXISTS templates (
     template_id      INTEGER PRIMARY KEY AUTOINCREMENT
    ,template_name    TEXT UNIQUE
    ,template_namespace TEXT COLLATE NOCASE
    ,template_data    BLOB
    ,template_created INTEGER
    ,template_updated INTEGER
);