DROP INDEX idx_linked_pullreqs_provider_id;
DROP TABLE linked_pullreqs;
ALTER TABLE pullreqs DROP COLUMN pullreq_type;
