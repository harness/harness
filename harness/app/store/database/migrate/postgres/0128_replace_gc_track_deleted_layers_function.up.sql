CREATE OR REPLACE FUNCTION gc_track_deleted_layers()
    RETURNS TRIGGER
AS
$$
BEGIN
    IF (TG_LEVEL = 'STATEMENT') THEN
        INSERT INTO gc_blob_review_queue (blob_id, review_after, event)
SELECT DISTINCT deleted_rows.layer_blob_id,
       gc_review_after('layer_delete'),
       'layer_delete'
FROM old_table deleted_rows
         JOIN
     blobs b ON deleted_rows.layer_blob_id = b.blob_id
ORDER BY deleted_rows.layer_blob_id ASC
    ON CONFLICT (blob_id)
            DO UPDATE SET review_after = gc_review_after('layer_delete'),
                       event        = 'layer_delete';
ELSIF (TG_LEVEL = 'ROW') THEN
        INSERT INTO gc_blob_review_queue (blob_id, review_after, event)
        VALUES (OLD.blob_id, gc_review_after('layer_delete'), 'layer_delete')
        ON CONFLICT (blob_id)
            DO UPDATE SET review_after = gc_review_after('layer_delete'),
                         event        = 'layer_delete';
END IF;
RETURN NULL;
END;
$$
LANGUAGE plpgsql;