ALTER TABLE checks ADD COLUMN check_type TEXT NOT NULL DEFAULT '';
ALTER TABLE checks DROP COLUMN check_payload_version;
ALTER TABLE checks DROP COLUMN check_payload_kind;
