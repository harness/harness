CREATE UNIQUE INDEX pullreqs_target_repo_branch_source_repo_branch
    ON pullreqs(pullreq_target_repo_id, pullreq_target_branch, pullreq_source_repo_id, pullreq_source_branch)
    WHERE pullreq_state = 'open';
