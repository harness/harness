CREATE INDEX pullreqs_target_repo_branch
    ON pullreqs (pullreq_target_repo_id, pullreq_target_branch)
    WHERE pullreq_state NOT IN ('closed', 'merged');
