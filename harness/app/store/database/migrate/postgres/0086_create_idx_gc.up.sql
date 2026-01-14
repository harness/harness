CREATE INDEX IF NOT EXISTS index_gc_manifest_review_queue2 
ON gc_manifest_review_queue (registry_id, manifest_id, review_after);
