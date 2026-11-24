ALTER TABLE pullreq_reviewer_suggestions
    ADD COLUMN pullreq_reviewer_suggestion_reason TEXT,
    ADD COLUMN pullreq_reviewer_suggestion_kind TEXT NOT NULL DEFAULT 'optional',
    ADD COLUMN pullreq_reviewer_suggestion_rank INTEGER,
    ADD COLUMN pullreq_reviewer_suggestion_user_group_id INTEGER,
    ADD COLUMN pullreq_reviewer_suggestion_commit_sha TEXT;

ALTER TABLE pullreq_reviewer_suggestions
    ADD CONSTRAINT fk_pullreq_reviewer_suggestions_user_group_id
        FOREIGN KEY (pullreq_reviewer_suggestion_user_group_id)
        REFERENCES usergroups (usergroup_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE SET NULL;
