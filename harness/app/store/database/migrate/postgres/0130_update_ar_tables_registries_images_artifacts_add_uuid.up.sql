CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

ALTER TABLE registries
    ADD COLUMN registry_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4();

ALTER TABLE images
    ADD COLUMN image_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4();

ALTER TABLE artifacts
    ADD COLUMN artifact_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4();