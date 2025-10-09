ALTER TABLE checks
    ADD COLUMN check_started BIGINT NOT NULL DEFAULT 0;
ALTER TABLE checks
    ADD COLUMN check_ended BIGINT NOT NULL DEFAULT 0;

UPDATE checks
SET check_started = check_created;

UPDATE checks
SET check_ended = check_updated;
