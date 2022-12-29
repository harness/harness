CREATE TABLE tokens (
 token_id             SERIAL PRIMARY KEY
,token_type           TEXT
,token_uid            TEXT
,token_principal_id   INTEGER
,token_expires_at     BIGINT
,token_grants         BIGINT
,token_issued_at      BIGINT
,token_created_by     INTEGER
,UNIQUE(token_principal_id, token_uid)
);
