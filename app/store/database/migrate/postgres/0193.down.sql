ALTER TABLE pullreq_reviewer_suggestions
    DROP CONSTRAINT fk_pullreq_reviewer_suggestions_user_group_id;

ALTER TABLE pullreq_reviewer_suggestions
    DROP COLUMN pullreq_reviewer_suggestion_commit_sha,
    DROP COLUMN pullreq_reviewer_suggestion_user_group_id,
    DROP COLUMN pullreq_reviewer_suggestion_rank,
    DROP COLUMN pullreq_reviewer_suggestion_kind,
    DROP COLUMN pullreq_reviewer_suggestion_reason;
