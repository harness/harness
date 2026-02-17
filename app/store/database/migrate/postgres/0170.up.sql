DELETE FROM artifacts
WHERE artifact_id IN (
    WITH valid_hex AS (
        SELECT
            a.artifact_id,
            CASE
                WHEN a.artifact_version ~ '^[0-9A-Fa-f]+$'
                AND length(a.artifact_version) % 2 = 0
    THEN decode(a.artifact_version, 'hex')
END AS artifact_digest
    FROM artifacts a
    JOIN images i ON i.image_id = a.artifact_image_id
    JOIN registries r ON r.registry_id = i.image_registry_id
    WHERE r.registry_package_type IN ('DOCKER', 'HELM')
  )
SELECT a.artifact_id
FROM artifacts a
         JOIN images i ON i.image_id = a.artifact_image_id
         JOIN registries r ON r.registry_id = i.image_registry_id
         LEFT JOIN valid_hex vh ON vh.artifact_id = a.artifact_id
         LEFT JOIN manifests m
                   ON m.manifest_registry_id = r.registry_id
                       AND m.manifest_image_name  = i.image_name
                       AND m.manifest_digest      = vh.artifact_digest
WHERE r.registry_package_type IN ('DOCKER', 'HELM')
  AND m.manifest_id IS NULL
    );