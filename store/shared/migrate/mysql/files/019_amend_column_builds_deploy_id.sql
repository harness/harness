-- name: alter-table-builds-alter-column-deploy-id

ALTER TABLE builds ALTER COLUMN build_deploy_id BIGINT NOT NULL DEFAULT 0;