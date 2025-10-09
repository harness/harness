CREATE TABLE public_key_sub_keys
(
    public_key_sub_key_public_key_id INTEGER NOT NULL,
    public_key_sub_key_id TEXT NOT NULL,

    CONSTRAINT pk_pgp_key_ids PRIMARY KEY (public_key_sub_key_public_key_id, public_key_sub_key_id),

    CONSTRAINT fk_public_key_sub_key_public_key_id FOREIGN KEY (public_key_sub_key_public_key_id)
        REFERENCES public_keys (public_key_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE INDEX idx_public_key_sub_key_id ON public_key_sub_keys(public_key_sub_key_id);

ALTER TABLE public_keys
    ADD COLUMN public_key_valid_from BIGINT,
    ADD COLUMN public_key_valid_to BIGINT,
    ADD COLUMN public_key_revocation_reason TEXT,
    ADD COLUMN public_key_metadata JSON NOT NULL DEFAULT '{}';
