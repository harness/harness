UPDATE pullreqs SET pullreq_merge_base_sha = '' WHERE pullreq_merge_base_sha IS NULL;
ALTER TABLE pullreqs
    ALTER COLUMN pullreq_merge_base_sha SET DEFAULT '',
    ALTER COLUMN pullreq_merge_base_sha SET NOT NULL;