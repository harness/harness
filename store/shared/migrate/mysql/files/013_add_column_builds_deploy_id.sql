-- name: alter-table-builds-add-column-deploy-id

ALTER TABLE builds ADD COLUMN build_deploy_id INTEGER NOT NULL DEFAULT 0;
