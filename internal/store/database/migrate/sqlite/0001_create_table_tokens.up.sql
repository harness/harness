CREATE TABLE IF NOT EXISTS tokens (
 token_id             INTEGER PRIMARY KEY AUTOINCREMENT
,token_type           TEXT COLLATE NOCASE
,token_name           TEXT
,token_principalId    INTEGER
,token_expiresAt      INTEGER
,token_grants         INTEGER
,token_issuedAt       INTEGER
,token_createdBy      INTEGER
);
