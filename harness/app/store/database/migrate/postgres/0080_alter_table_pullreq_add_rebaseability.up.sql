ALTER TABLE pullreqs
    ADD COLUMN pullreq_rebase_check_status TEXT NOT NULL DEFAULT 'unchecked',
    ADD COLUMN pullreq_rebase_conflicts TEXT;
