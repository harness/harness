CREATE UNIQUE INDEX pullreqs_source_repo_branch_target_repo_branch
    ON pullreqs(pullreq_source_repo_id, pullreq_source_branch, pullreq_target_repo_id, pullreq_target_branch)
    WHERE pullreq_state = 'open';
