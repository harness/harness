ALTER TABLE checks
    ADD COLUMN check_started INTEGER NOT NULL DEFAULT 0;
ALTER TABLE checks
    ADD COLUMN check_ended INTEGER NOT NULL DEFAULT 0;

UPDATE checks
SET check_started = check_created;

UPDATE checks
SET check_ended = check_updated;
