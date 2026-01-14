ALTER TABLE checks ADD COLUMN check_payload_version TEXT NOT NULL DEFAULT '';
ALTER TABLE checks ADD COLUMN check_payload_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE checks DROP COLUMN check_type;
