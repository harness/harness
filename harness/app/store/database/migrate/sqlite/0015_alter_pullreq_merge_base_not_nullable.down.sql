ALTER TABLE pullreqs ADD COLUMN pullreq_merge_base_sha_nullable TEXT;
UPDATE pullreqs SET pullreq_merge_base_sha_nullable = pullreq_merge_base_sha WHERE pullreq_merge_base_sha <> '';
ALTER TABLE pullreqs DROP COLUMN pullreq_merge_base_sha;
ALTER TABLE pullreqs RENAME COLUMN pullreq_merge_base_sha_nullable TO pullreq_merge_base_sha;
