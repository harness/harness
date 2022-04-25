PRAGMA foreign_keys=off;

BEGIN TRANSACTION;

ALTER TABLE templates RENAME TO _templates_old;

CREATE TABLE IF NOT EXISTS templates (
     template_id      INTEGER PRIMARY KEY AUTOINCREMENT
    ,template_name    TEXT
    ,template_namespace TEXT COLLATE NOCASE
    ,template_data    BLOB
    ,template_created INTEGER
    ,template_updated INTEGER
,UNIQUE(template_name, template_namespace)
);

INSERT INTO templates (template_id, template_name, template_namespace, template_data, template_created, template_updated)
  SELECT template_id, template_name, template_namespace, template_data, template_created, template_updated
  FROM _templates_old;

COMMIT;

CREATE INDEX IF NOT EXISTS ix_template_namespace ON templates (template_namespace);
DROP TABLE _templates_old;

PRAGMA foreign_keys=on;
