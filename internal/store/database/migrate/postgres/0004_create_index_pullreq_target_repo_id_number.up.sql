CREATE UNIQUE INDEX IF NOT EXISTS index_pullreq_target_repo_id_number
ON pullreq(pullreq_target_repo_id, pullreq_number);
