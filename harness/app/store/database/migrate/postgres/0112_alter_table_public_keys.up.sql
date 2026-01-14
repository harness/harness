ALTER TABLE public_keys
    ADD COLUMN public_key_scheme TEXT DEFAULT 'ssh' NOT NULL;