ALTER TABLE webhooks
    ADD COLUMN webhook_type INTEGER DEFAULT 0;

UPDATE webhooks SET webhook_type = 1 WHERE webhook_internal = TRUE;

ALTER TABLE webhooks DROP COLUMN webhook_internal;
