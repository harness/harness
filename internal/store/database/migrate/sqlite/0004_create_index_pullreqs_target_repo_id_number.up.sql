CREATE UNIQUE INDEX pullreqs_target_repo_id_number
ON pullreqs(pullreq_target_repo_id, pullreq_number);
