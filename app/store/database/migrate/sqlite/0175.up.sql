CREATE TABLE repo_activities (
 repo_activity_key TEXT PRIMARY KEY
,repo_activity_repo_id BIGINT NOT NULL
,repo_activity_principal_id BIGINT NOT NULL
,repo_activity_type TEXT NOT NULL
,repo_activity_payload TEXT NOT NULL DEFAULT '{}'
,repo_activity_created_at BIGINT NOT NULL
,CONSTRAINT fk_repo_activities_repo_id FOREIGN KEY (repo_activity_repo_id)
    REFERENCES repositories (repo_id)
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_repo_activities_principal_id FOREIGN KEY (repo_activity_principal_id)
    REFERENCES principals (principal_id)
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
);

CREATE INDEX repo_activities_repo_id_created_at_type_idx
    ON repo_activities (repo_activity_repo_id, repo_activity_created_at, repo_activity_type);
