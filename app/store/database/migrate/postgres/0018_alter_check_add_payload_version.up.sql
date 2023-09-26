ALTER TABLE checks
    ADD COLUMN check_payload_version TEXT NOT NULL DEFAULT '',
    ADD COLUMN check_payload_kind TEXT NOT NULL DEFAULT '',
    DROP COLUMN check_type;
