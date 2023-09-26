CREATE TABLE pullreqs (
pullreq_id INTEGER PRIMARY KEY AUTOINCREMENT
,pullreq_version INTEGER NOT NULL DEFAULT 0
,pullreq_created_by INTEGER NOT NULL
,pullreq_created BIGINT NOT NULL
,pullreq_updated BIGINT NOT NULL
,pullreq_edited BIGINT NOT NULL
,pullreq_number INTEGER NOT NULL
,pullreq_state TEXT NOT NULL
,pullreq_is_draft TEXT NOT NULL DEFAULT FALSE
,pullreq_comment_count INTEGER NOT NULL DEFAULT 0
,pullreq_title TEXT NOT NULL
,pullreq_description TEXT NOT NULL
,pullreq_source_repo_id INTEGER NOT NULL
,pullreq_source_branch TEXT NOT NULL
,pullreq_source_sha TEXT NOT NULL
,pullreq_target_repo_id INTEGER NOT NULL
,pullreq_target_branch TEXT NOT NULL
,pullreq_activity_seq INTEGER DEFAULT 0
,pullreq_merged_by INTEGER
,pullreq_merged BIGINT
,pullreq_merge_method TEXT
,pullreq_merge_check_status TEXT NOT NULL
,pullreq_merge_target_sha TEXT
,pullreq_merge_base_sha TEXT
,pullreq_merge_sha TEXT
,pullreq_merge_conflicts TEXT
,CONSTRAINT fk_pullreq_created_by FOREIGN KEY (pullreq_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
,CONSTRAINT fk_pullreq_source_repo_id FOREIGN KEY (pullreq_source_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE SET NULL
,CONSTRAINT fk_pullreq_target_repo_id FOREIGN KEY (pullreq_target_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_merged_by FOREIGN KEY (pullreq_merged_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
);
