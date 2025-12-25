-- Drop tables
DROP TABLE IF EXISTS registry_blobs;
DROP TABLE IF EXISTS manifest_references;
DROP TABLE IF EXISTS layers;
DROP TABLE IF EXISTS artifact_stats;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS upstream_proxy_configs;
DROP TABLE IF EXISTS cleanup_policy_prefix_mappings;
DROP TABLE IF EXISTS cleanup_policies;
DROP TABLE IF EXISTS gc_blob_review_queue;
DROP TABLE IF EXISTS gc_review_after_defaults;
DROP TABLE IF EXISTS gc_manifest_review_queue;
DROP TABLE IF EXISTS manifests;
DROP TABLE IF EXISTS registries;
DROP TABLE IF EXISTS blobs;
DROP TABLE IF EXISTS media_types;

-- Drop functions
DROP FUNCTION IF EXISTS gc_review_after(text);
DROP FUNCTION IF EXISTS gc_track_blob_uploads();
DROP FUNCTION IF EXISTS gc_track_manifest_uploads();
DROP FUNCTION IF EXISTS gc_track_deleted_manifests();
DROP FUNCTION IF EXISTS gc_track_deleted_layers();
DROP FUNCTION IF EXISTS gc_track_deleted_manifest_lists();
DROP FUNCTION IF EXISTS gc_track_deleted_tags();
DROP FUNCTION IF EXISTS gc_track_switched_tags();