DROP FUNCTION IF EXISTS gc_track_deleted_manifest_lists CASCADE;

CREATE
OR REPLACE FUNCTION gc_track_deleted_manifest_lists()
    RETURNS TRIGGER
AS
$$
BEGIN
INSERT INTO gc_manifest_review_queue (registry_id, manifest_id, review_after, event)
VALUES (OLD.manifest_ref_registry_id, OLD.manifest_ref_child_id, gc_review_after('manifest_list_delete'),
        'manifest_list_delete') ON CONFLICT (registry_id, manifest_id)
        DO
UPDATE SET review_after = gc_review_after('manifest_list_delete'),
    event = 'manifest_list_delete';
RETURN NULL;
END;
$$
LANGUAGE plpgsql;