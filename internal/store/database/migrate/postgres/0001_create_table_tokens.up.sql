CREATE TABLE IF NOT EXISTS tokens (
 token_id             SERIAL PRIMARY KEY
,token_type           TEXT
,token_name           TEXT
,token_principalId    INTEGER
,token_expiresAt      BIGINT
,token_grants         BIGINT
,token_issuedAt       BIGINT
,token_createdBy      INTEGER
);
