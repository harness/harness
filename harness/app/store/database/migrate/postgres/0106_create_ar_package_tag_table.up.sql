CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE package_tags (
                              package_tag_id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                              package_tag_name            TEXT    NOT NULL,
                              package_tag_artifact_id     BIGINT REFERENCES artifacts (artifact_id)  ON DELETE CASCADE,
                              package_tag_created_at      BIGINT NOT NULL,
                              package_tag_created_by      INTEGER NOT NULL,
                              package_tag_updated_at      BIGINT NOT NULL,
                              package_tag_updated_by      INTEGER NOT NULL,
                              CONSTRAINT unique_package_tag UNIQUE (package_tag_name, package_tag_artifact_id)
);
