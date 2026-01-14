ALTER TABLE pullreqs
    ALTER COLUMN pullreq_merge_base_sha DROP DEFAULT,
    ALTER COLUMN pullreq_merge_base_sha DROP NOT NULL;
UPDATE pullreqs SET pullreq_merge_base_sha = NULL WHERE pullreq_merge_base_sha = '';