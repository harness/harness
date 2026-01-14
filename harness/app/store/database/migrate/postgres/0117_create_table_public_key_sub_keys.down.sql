ALTER TABLE public_keys
    DROP COLUMN public_key_valid_from,
    DROP COLUMN public_key_valid_to,
    DROP COLUMN public_key_revocation_reason,
    DROP COLUMN public_key_metadata;

DROP INDEX idx_public_key_sub_key_id;

DROP TABLE public_key_sub_keys;
