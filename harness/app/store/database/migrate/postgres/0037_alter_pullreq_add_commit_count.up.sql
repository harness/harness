ALTER TABLE pullreqs
    ADD COLUMN pullreq_commit_count INTEGER,
    ADD COLUMN pullreq_file_count INTEGER;