-- name: create-table-templates

CREATE TABLE IF NOT EXISTS templates (
     template_id      INTEGER PRIMARY KEY AUTOINCREMENT
    ,template_name    TEXT UNIQUE
    ,template_namespace TEXT COLLATE NOCASE
    ,template_data    BLOB
    ,template_created INTEGER
    ,template_updated INTEGER
,UNIQUE(template_name, template_namespace)
);

CREATE INDEX IF NOT EXISTS ix_template_namespace ON templates (template_namespace);
