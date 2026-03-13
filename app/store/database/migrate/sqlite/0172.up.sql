DELETE FROM registry_blobs
WHERE rblob_id IN (
    SELECT rb.rblob_id
    FROM registry_blobs rb
             LEFT JOIN images i ON rb.rblob_registry_id = i.image_registry_id
        AND rb.rblob_image_name = i.image_name
    WHERE i.image_id IS NULL
);