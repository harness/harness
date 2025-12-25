ALTER TABLE pullreqs ADD COLUMN pullreq_merge_base_sha_not_nullable TEXT NOT NULL DEFAULT '';
UPDATE pullreqs SET pullreq_merge_base_sha_not_nullable = pullreq_merge_base_sha WHERE pullreq_merge_base_sha IS NOT NULL;
ALTER TABLE pullreqs DROP COLUMN pullreq_merge_base_sha;
ALTER TABLE pullreqs RENAME COLUMN pullreq_merge_base_sha_not_nullable TO pullreq_merge_base_sha;
