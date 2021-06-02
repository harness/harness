-- name: create-table-template

CREATE TABLE IF NOT EXISTS templates (
    template_id       SERIAL PRIMARY KEY
    ,template_name    TEXT UNIQUE
    ,template_namespace VARCHAR(50)
    ,template_data    BYTEA
    ,template_created INTEGER
    ,template_updated INTEGER
,UNIQUE(template_name, template_namespace)
);

CREATE INDEX IF NOT EXISTS ix_template_namespace ON templates (template_namespace);
