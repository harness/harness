ALTER TABLE gitspace_events RENAME COLUMN geven_entity_uid to geven_query_key;

ALTER TABLE gitspace_events
    ADD COLUMN geven_timestamp BIGINT;
UPDATE gitspace_events
SET geven_timestamp = geven_created * 1000000;
ALTER TABLE gitspace_events
    ALTER COLUMN geven_timestamp SET NOT NULL;