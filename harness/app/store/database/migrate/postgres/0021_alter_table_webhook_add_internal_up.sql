ALTER TABLE webhooks
    ADD COLUMN webhook_internal BOOLEAN NOT NULL DEFAULT false;
