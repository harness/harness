ALTER TABLE gitspace_events RENAME COLUMN geven_query_key to geven_entity_uid;

ALTER TABLE gitspace_events DROP COLUMN geven_timestamp;