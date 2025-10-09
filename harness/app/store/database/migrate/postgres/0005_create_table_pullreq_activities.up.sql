CREATE TABLE pullreq_activities (
 pullreq_activity_id SERIAL PRIMARY KEY
,pullreq_activity_version BIGINT NOT NULL
,pullreq_activity_created_by INTEGER
,pullreq_activity_created BIGINT NOT NULL
,pullreq_activity_updated BIGINT NOT NULL
,pullreq_activity_edited BIGINT NOT NULL
,pullreq_activity_deleted BIGINT
,pullreq_activity_parent_id INTEGER
,pullreq_activity_repo_id INTEGER NOT NULL
,pullreq_activity_pullreq_id INTEGER NOT NULL
,pullreq_activity_order INTEGER NOT NULL
,pullreq_activity_sub_order INTEGER NOT NULL
,pullreq_activity_reply_seq INTEGER NOT NULL
,pullreq_activity_type TEXT NOT NULL
,pullreq_activity_kind TEXT NOT NULL
,pullreq_activity_text TEXT NOT NULL
,pullreq_activity_payload JSONB NOT NULL DEFAULT '{}'
,pullreq_activity_metadata JSONB NOT NULL DEFAULT '{}'
,pullreq_activity_resolved_by INTEGER DEFAULT 0
,pullreq_activity_resolved BIGINT NULL
,CONSTRAINT fk_pullreq_activities_created_by FOREIGN KEY (pullreq_activity_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
,CONSTRAINT fk_pullreq_activities_parent_id FOREIGN KEY (pullreq_activity_parent_id)
    REFERENCES pullreq_activities (pullreq_activity_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_activities_repo_id FOREIGN KEY (pullreq_activity_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_activities_pullreq_id FOREIGN KEY (pullreq_activity_pullreq_id)
    REFERENCES pullreqs (pullreq_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_activities_resolved_by FOREIGN KEY (pullreq_activity_resolved_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
);
