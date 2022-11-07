CREATE TABLE IF NOT EXISTS tokens (
 token_id             INTEGER PRIMARY KEY AUTOINCREMENT
,token_type           TEXT COLLATE NOCASE
,token_uid            TEXT COLLATE NOCASE
,token_principalId    INTEGER
,token_expiresAt      BIGINT
,token_grants         BIGINT
,token_issuedAt       BIGINT
,token_createdBy      INTEGER
,UNIQUE(token_principalId, token_uid COLLATE NOCASE)
);
