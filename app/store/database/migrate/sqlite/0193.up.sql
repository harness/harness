ALTER TABLE pullreq_reviewer_suggestions ADD COLUMN pullreq_reviewer_suggestion_reason TEXT;
ALTER TABLE pullreq_reviewer_suggestions ADD COLUMN pullreq_reviewer_suggestion_kind TEXT NOT NULL DEFAULT 'optional';
ALTER TABLE pullreq_reviewer_suggestions ADD COLUMN pullreq_reviewer_suggestion_rank INTEGER;
ALTER TABLE pullreq_reviewer_suggestions ADD COLUMN pullreq_reviewer_suggestion_user_group_id INTEGER
    REFERENCES usergroups (usergroup_id) ON DELETE SET NULL;
ALTER TABLE pullreq_reviewer_suggestions ADD COLUMN pullreq_reviewer_suggestion_commit_sha TEXT;
