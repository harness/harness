CREATE TABLE public_keys (
 public_key_id INTEGER PRIMARY KEY AUTOINCREMENT
,public_key_principal_id INTEGER NOT NULL
,public_key_created BIGINT NOT NULL
,public_key_verified BIGINT
,public_key_identifier TEXT NOT NULL
,public_key_usage TEXT NOT NULL
,public_key_fingerprint TEXT NOT NULL
,public_key_content TEXT NOT NULL
,public_key_comment TEXT NOT NULL
,public_key_type TEXT NOT NULL
,CONSTRAINT fk_public_key_principal_id FOREIGN KEY (public_key_principal_id)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);

CREATE INDEX public_keys_fingerprint
    ON public_keys(public_key_fingerprint);

CREATE UNIQUE INDEX public_keys_principal_id_identifier
    ON public_keys(public_key_principal_id, LOWER(public_key_identifier));
