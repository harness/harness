ALTER TABLE download_stats DROP CONSTRAINT IF EXISTS fk_artifacts_artifact_id;

ALTER TABLE download_stats
ADD CONSTRAINT fk_artifacts_artifact_id
FOREIGN KEY (download_stat_artifact_id) 
REFERENCES artifacts(artifact_id)
ON DELETE CASCADE;