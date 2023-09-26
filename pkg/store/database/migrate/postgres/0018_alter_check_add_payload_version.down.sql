ALTER TABLE checks
    ADD COLUMN check_type TEXT NOT NULL DEFAULT '',
    DROP COLUMN check_payload_version,
    DROP COLUMN check_payload_kind;
