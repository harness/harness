CREATE UNIQUE INDEX IF NOT EXISTS index_pullreq_target_repo_id_number
ON pullreqs(pullreq_target_repo_id, pullreq_number);
