-- name: create-table-template

CREATE TABLE IF NOT EXISTS templates (
     template_id      INTEGER PRIMARY KEY AUTO_INCREMENT
    ,template_name    VARCHAR(500)
    ,template_namespace VARCHAR(50)
    ,template_data    BLOB
    ,template_created INTEGER
    ,template_updated INTEGER
    ,UNIQUE(template_name, template_namespace)
    );

-- name: create-index-template-namespace

CREATE INDEX ix_template_namespace ON templates (template_namespace);