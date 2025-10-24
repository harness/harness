DROP INDEX pullreqs_source_repo_branch_target_repo_branch;

ALTER TABLE pullreqs
    DROP CONSTRAINT fk_pullreq_source_repo_id;

ALTER TABLE pullreqs
    ALTER COLUMN pullreq_source_repo_id DROP NOT NULL;

ALTER TABLE pullreqs
    ALTER COLUMN pullreq_activity_seq SET NOT NULL;

ALTER TABLE pullreqs
    ALTER COLUMN pullreq_merge_check_status SET DEFAULT 'unchecked'::text;

ALTER TABLE pullreqs
    ADD CONSTRAINT fk_pullreq_source_repo_id FOREIGN KEY (pullreq_source_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE SET NULL;

CREATE UNIQUE INDEX pullreqs_source_repo_branch_target_repo_branch
    ON pullreqs (pullreq_source_repo_id, pullreq_source_branch, pullreq_target_repo_id, pullreq_target_branch)
    WHERE pullreq_state = 'open' and pullreq_source_repo_id IS NOT NULL;
