CREATE INDEX webhooks_space_id ON webhooks(webhook_space_id);
CREATE INDEX webhooks_repo_id ON webhooks(webhook_repo_id);

DROP INDEX webhooks_space_id_uid;
DROP INDEX webhooks_repo_id_uid;
ALTER TABLE webhooks DROP COLUMN webhook_uid;