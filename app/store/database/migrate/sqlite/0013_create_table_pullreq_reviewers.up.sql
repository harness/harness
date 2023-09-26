CREATE TABLE pullreq_reviewers (
pullreq_reviewer_pullreq_id INTEGER NOT NULL
,pullreq_reviewer_principal_id INTEGER NOT NULL
,pullreq_reviewer_created_by INTEGER NOT NULL
,pullreq_reviewer_created BIGINT NOT NULL
,pullreq_reviewer_updated BIGINT NOT NULL
,pullreq_reviewer_repo_id INTEGER NOT NULL
,pullreq_reviewer_type TEXT NOT NULL
,pullreq_reviewer_latest_review_id INTEGER
,pullreq_reviewer_review_decision TEXT NOT NULL
,pullreq_reviewer_sha TEXT NOT NULL
,CONSTRAINT pk_pullreq_reviewers PRIMARY KEY (pullreq_reviewer_pullreq_id, pullreq_reviewer_principal_id)
,CONSTRAINT fk_pullreq_reviewer_pullreq_id FOREIGN KEY (pullreq_reviewer_pullreq_id)
    REFERENCES pullreqs (pullreq_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_reviewer_user_id FOREIGN KEY (pullreq_reviewer_principal_id)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
,CONSTRAINT fk_pullreq_reviewer_created_by FOREIGN KEY (pullreq_reviewer_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
,CONSTRAINT fk_pullreq_reviewer_repo_id FOREIGN KEY (pullreq_reviewer_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_reviewer_latest_review_id FOREIGN KEY (pullreq_reviewer_latest_review_id)
    REFERENCES pullreq_reviews (pullreq_review_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE SET NULL
);