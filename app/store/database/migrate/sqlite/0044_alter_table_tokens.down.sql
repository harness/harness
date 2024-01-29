-- recreate original table with UNIQUE
CREATE TABLE tokens_new (
 token_id             INTEGER PRIMARY KEY AUTOINCREMENT
,token_type           TEXT COLLATE NOCASE
,token_uid            TEXT COLLATE NOCASE
,token_principal_id   INTEGER
,token_expires_at     BIGINT
,token_issued_at      BIGINT
,token_created_by     INTEGER
,UNIQUE(token_principal_id, token_uid COLLATE NOCASE)

,CONSTRAINT fk_token_principal_id FOREIGN KEY (token_principal_id)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);

-- copy over data
INSERT INTO tokens_new(
    token_id
    ,token_type
    ,token_uid
    ,token_principal_id
    ,token_expires_at
    ,token_issued_at
    ,token_created_by
)
SELECT
    token_id
    ,token_type
    ,token_uid
    ,token_principal_id
    ,token_expires_at
    ,token_issued_at
    ,token_created_by
FROM tokens;

-- delete old table (also deletes all indices)
DROP TABLE tokens;

-- rename table
ALTER TABLE tokens_new RENAME TO tokens;

-- recreate all previous indices
CREATE INDEX tokens_principal_id ON tokens(token_principal_id);
CREATE INDEX tokens_type_expires_at ON tokens(token_type, token_expires_at);

