-- +migrate Up

ALTER TABLE repos ADD COLUMN repo_gated BOOLEAN;
UPDATE repos SET repo_gated = false;

CREATE TABLE senders (
 sender_id      SERIAL PRIMARY KEY
,sender_repo_id INTEGER
,sender_login   VARCHAR(250)
,sender_allow   BOOLEAN
,sender_block   BOOLEAN

,UNIQUE(sender_repo_id,sender_login)
);

CREATE INDEX sender_repo_ix ON senders (sender_repo_id);

-- +migrate Down

ALTER TABLE repos DROP COLUMN repo_gated;
DROP INDEX sender_repo_ix;
DROP TABLE senders;
