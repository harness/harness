CREATE TABLE package_tags (
                              package_tag_id              TEXT PRIMARY KEY,
                              package_tag_name            TEXT    NOT NULL,
                              package_tag_artifact_id     INTEGER REFERENCES artifacts (artifact_id)  ON DELETE CASCADE,
                              package_tag_created_at      INTEGER NOT NULL,
                              package_tag_created_by      INTEGER NOT NULL,
                              package_tag_updated_at      INTEGER NOT NULL,
                              package_tag_updated_by      INTEGER NOT NULL,
                              CONSTRAINT unique_package_tag UNIQUE (package_tag_name, package_tag_artifact_id)
);
