CREATE TABLE IF NOT EXISTS tokens (
 token_id             SERIAL PRIMARY KEY
,token_type           TEXT
,token_name           TEXT
,token_principalId    INTEGER
,token_expiresAt      INTEGER
,token_grants         INTEGER
,token_issuedAt       INTEGER
,token_createdBy      INTEGER
);
