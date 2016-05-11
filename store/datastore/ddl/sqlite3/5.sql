-- +migrate Up

ALTER TABLE builds ADD COLUMN build_signed   BOOLEAN;
ALTER TABLE builds ADD COLUMN build_verified BOOLEAN;

UPDATE builds SET build_signed   = 0;
UPDATE builds SET build_verified = 0;

-- +migrate Down

ALTER TABLE builds DROP COLUMN build_signed;
ALTER TABLE builds DROP COLUMN build_verified;
