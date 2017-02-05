-- +migrate Up

ALTER TABLE builds ADD COLUMN build_parent INTEGER DEFAULT 0;

-- +migrate Down

ALTER TABLE builds DROP COLUMN build_parent;
