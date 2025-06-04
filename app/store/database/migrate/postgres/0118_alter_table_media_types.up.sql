ALTER TABLE media_types
    ADD COLUMN is_runnable BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN media_types.is_runnable IS
    'Indicates whether artifacts with this media type can be pulled using Docker client or similar runtime.
    If false, scanning will be skipped as the artifact is not runnable via standard Docker tools.
    Useful for distinguishing signature/attestation media types (e.g., cosign) from executable image layers.';

INSERT INTO media_types (mt_media_type, is_runnable) 
VALUES ('application/vnd.dev.cosign.simplesigning.v1+json', false),
       ('application/vnd.dsse.envelope.v1+json', false) 
ON CONFLICT (mt_media_type)
DO NOTHING;
