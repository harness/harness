-- delete old unique constraint
ALTER TABLE tokens DROP CONSTRAINT tokens_token_principal_id_token_uid_key;

-- delete unnecessary index
DROP INDEX tokens_principal_id;

-- delete all duplicates tokens by keeping newest one (worst case user can recreate their old tokens)
DELETE FROM
    tokens t1
USING tokens t2
WHERE
    t1.token_id < t2.token_id
    AND t1.token_principal_id = t2.token_principal_id AND LOWER(t1.token_uid) = LOWER(t2.token_uid);

-- create explicit unique index with case insensitivity
CREATE UNIQUE INDEX tokens_principal_id_uid ON tokens(token_principal_id, LOWER(token_uid));
