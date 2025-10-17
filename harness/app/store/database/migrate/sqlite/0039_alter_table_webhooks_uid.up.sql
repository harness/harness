ALTER TABLE webhooks ADD COLUMN webhook_uid TEXT;

CREATE UNIQUE INDEX webhooks_repo_id_uid
    ON webhooks(webhook_repo_id, LOWER(webhook_uid))
    WHERE webhook_space_id IS NULL;

CREATE UNIQUE INDEX webhooks_space_id_uid
    ON webhooks(webhook_space_id, LOWER(webhook_uid))
    WHERE webhook_repo_id IS NULL;


DROP INDEX webhooks_repo_id;
DROP INDEX webhooks_space_id;

-- code migration will backfill uids