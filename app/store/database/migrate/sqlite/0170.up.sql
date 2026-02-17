DELETE FROM artifacts
WHERE EXISTS (
    SELECT 1
    FROM images i
             JOIN registries r
                  ON r.registry_id = i.image_registry_id
    WHERE i.image_id = artifacts.artifact_image_id
      AND r.registry_package_type IN ('DOCKER', 'HELM')
      AND NOT EXISTS (
        SELECT 1
        FROM manifests m
        WHERE m.manifest_registry_id = r.registry_id
          AND m.manifest_image_name  = i.image_name
          AND m.manifest_digest =
              CASE
                  WHEN length(artifacts.artifact_version) % 2 = 0
          AND lower(artifacts.artifact_version) GLOB '[0-9a-f]*'
            AND lower(artifacts.artifact_version) NOT GLOB '*[^0-9a-f]*'
        THEN unhex(artifacts.artifact_version)
        END
        )
);
