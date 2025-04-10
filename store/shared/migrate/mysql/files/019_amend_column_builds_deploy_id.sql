-- name: alter-table-builds-alter-column-deploy-id
ALTER TABLE builds
ALTER COLUMN build_deploy_id SET DATA TYPE BIGINT,
ALTER COLUMN build_deploy_id SET NOT NULL,
ALTER COLUMN build_deploy_id SET DEFAULT 0;