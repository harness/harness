ALTER TABLE pullreq_activities
    ADD COLUMN pullreq_activity_outdated BOOLEAN,
    ADD COLUMN pullreq_activity_code_comment_merge_base_sha TEXT,
    ADD COLUMN pullreq_activity_code_comment_source_sha TEXT,
    ADD COLUMN pullreq_activity_code_comment_path TEXT,
    ADD COLUMN pullreq_activity_code_comment_line_new INTEGER,
    ADD COLUMN pullreq_activity_code_comment_span_new INTEGER,
    ADD COLUMN pullreq_activity_code_comment_line_old INTEGER,
    ADD COLUMN pullreq_activity_code_comment_span_old INTEGER;