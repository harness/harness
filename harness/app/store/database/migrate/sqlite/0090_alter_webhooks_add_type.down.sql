ALTER TABLE webhooks
    ADD COLUMN webhook_internal BOOLEAN DEFAULT FALSE;

UPDATE webhooks SET webhook_internal = TRUE WHERE webhook_type = 1;

ALTER TABLE webhooks DROP COLUMN webhook_type;
