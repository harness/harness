-- drop new index
DROP INDEX tokens_principal_id_uid;

-- recreate unique constraint
ALTER TABLE tokens ADD CONSTRAINT tokens_token_principal_id_token_uid_key UNIQUE (token_principal_id, token_uid);

-- recreate original indices
CREATE INDEX tokens_principal_id ON tokens(token_principal_id);