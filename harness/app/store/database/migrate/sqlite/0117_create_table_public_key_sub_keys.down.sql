ALTER TABLE public_keys DROP COLUMN public_key_valid_from;
ALTER TABLE public_keys DROP COLUMN public_key_valid_to;
ALTER TABLE public_keys DROP COLUMN public_key_revocation_reason;
ALTER TABLE public_keys DROP COLUMN public_key_metadata;

DROP INDEX idx_public_key_sub_key_id;

DROP TABLE public_key_sub_keys;
