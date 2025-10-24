DROP INDEX pullreqs_source_repo_branch_target_repo_branch;
DROP INDEX pullreqs_target_repo_id_number;

CREATE TABLE pullreqs_tmp(
 pullreq_id INTEGER PRIMARY KEY AUTOINCREMENT
,pullreq_version INTEGER DEFAULT 0 NOT NULL
,pullreq_created_by INTEGER NOT NULL
,pullreq_created BIGINT NOT NULL
,pullreq_updated BIGINT NOT NULL
,pullreq_edited BIGINT NOT NULL
,pullreq_number INTEGER NOT NULL
,pullreq_state TEXT  NOT NULL
,pullreq_is_draft TEXT DEFAULT FALSE NOT NULL
,pullreq_comment_count INTEGER DEFAULT 0 NOT NULL
,pullreq_title TEXT NOT NULL
,pullreq_description TEXT NOT NULL
,pullreq_source_repo_id INTEGER
,pullreq_source_branch TEXT NOT NULL
,pullreq_source_sha TEXT NOT NULL
,pullreq_target_repo_id INTEGER NOT NULL
,pullreq_target_branch TEXT NOT NULL
,pullreq_activity_seq INTEGER NOT NULL DEFAULT 0
,pullreq_merged_by INTEGER
,pullreq_merged BIGINT
,pullreq_merge_method TEXT
,pullreq_merge_check_status TEXT DEFAULT 'unchecked' NOT NULL
,pullreq_merge_target_sha TEXT
,pullreq_merge_sha TEXT
,pullreq_merge_conflicts TEXT
,pullreq_merge_base_sha TEXT DEFAULT '' NOT NULL
,pullreq_unresolved_count INTEGER DEFAULT 0 NOT NULL
,pullreq_commit_count INTEGER
,pullreq_file_count INTEGER
,pullreq_closed BIGINT
,pullreq_additions INTEGER
,pullreq_deletions INTEGER
,pullreq_rebase_check_status TEXT DEFAULT 'unchecked' NOT NULL
,pullreq_rebase_conflicts TEXT
,pullreq_merge_violations_bypassed BOOLEAN default NULL
,CONSTRAINT fk_pullreq_created_by FOREIGN KEY (pullreq_created_by)
    REFERENCES principals
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
,CONSTRAINT fk_pullreq_source_repo_id FOREIGN KEY (pullreq_source_repo_id)
    REFERENCES repositories
    ON UPDATE NO ACTION
    ON DELETE SET NULL
,CONSTRAINT fk_pullreq_target_repo_id FOREIGN KEY (pullreq_target_repo_id)
    REFERENCES repositories
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_merged_by FOREIGN KEY (pullreq_merged_by)
    REFERENCES principals
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
);

INSERT INTO pullreqs_tmp(
    pullreq_id,
    pullreq_version,
    pullreq_created_by,
    pullreq_created,
    pullreq_updated,
    pullreq_edited,
    pullreq_number,
    pullreq_state,
    pullreq_is_draft,
    pullreq_comment_count,
    pullreq_title,
    pullreq_description,
    pullreq_source_repo_id,
    pullreq_source_branch,
    pullreq_source_sha,
    pullreq_target_repo_id,
    pullreq_target_branch,
    pullreq_activity_seq,
    pullreq_merged_by,
    pullreq_merged,
    pullreq_merge_method,
    pullreq_merge_check_status,
    pullreq_merge_target_sha,
    pullreq_merge_sha,
    pullreq_merge_conflicts,
    pullreq_merge_base_sha,
    pullreq_unresolved_count,
    pullreq_commit_count,
    pullreq_file_count,
    pullreq_closed,
    pullreq_additions,
    pullreq_deletions,
    pullreq_rebase_check_status,
    pullreq_rebase_conflicts,
    pullreq_merge_violations_bypassed
)
SELECT
    pullreq_id,
    pullreq_version,
    pullreq_created_by,
    pullreq_created,
    pullreq_updated,
    pullreq_edited,
    pullreq_number,
    pullreq_state,
    pullreq_is_draft,
    pullreq_comment_count,
    pullreq_title,
    pullreq_description,
    pullreq_source_repo_id,
    pullreq_source_branch,
    pullreq_source_sha,
    pullreq_target_repo_id,
    pullreq_target_branch,
    pullreq_activity_seq,
    pullreq_merged_by,
    pullreq_merged,
    pullreq_merge_method,
    pullreq_merge_check_status,
    pullreq_merge_target_sha,
    pullreq_merge_sha,
    pullreq_merge_conflicts,
    pullreq_merge_base_sha,
    pullreq_unresolved_count,
    pullreq_commit_count,
    pullreq_file_count,
    pullreq_closed,
    pullreq_additions,
    pullreq_deletions,
    pullreq_rebase_check_status,
    pullreq_rebase_conflicts,
    pullreq_merge_violations_bypassed
FROM pullreqs;

DROP TABLE pullreqs;

ALTER TABLE pullreqs_tmp RENAME TO pullreqs;

CREATE UNIQUE INDEX pullreqs_target_repo_id_number
    on pullreqs (pullreq_target_repo_id, pullreq_number);

CREATE UNIQUE INDEX pullreqs_source_repo_branch_target_repo_branch
    ON pullreqs (pullreq_source_repo_id, pullreq_source_branch, pullreq_target_repo_id, pullreq_target_branch)
    WHERE pullreq_state = 'open' and pullreq_source_repo_id IS NOT NULL;
