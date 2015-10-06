-- +migrate Up

CREATE TABLE users (
 user_id     INTEGER PRIMARY KEY AUTOINCREMENT
,user_login  TEXT
,user_token  TEXT
,user_secret TEXT
,user_expiry INTEGER
,user_email  TEXT
,user_avatar TEXT
,user_active BOOLEAN
,user_admin  BOOLEAN
,user_hash   TEXT

,UNIQUE(user_login)
);

CREATE TABLE repos (
 repo_id            INTEGER PRIMARY KEY AUTOINCREMENT
,repo_user_id       INTEGER 
,repo_owner         TEXT
,repo_name          TEXT
,repo_full_name     TEXT
,repo_avatar        TEXT
,repo_link          TEXT
,repo_clone         TEXT
,repo_branch        TEXT
,repo_timeout       INTEGER
,repo_private       BOOLEAN
,repo_trusted       BOOLEAN
,repo_allow_pr      BOOLEAN
,repo_allow_push    BOOLEAN
,repo_allow_deploys BOOLEAN
,repo_allow_tags    BOOLEAN
,repo_hash          TEXT

,UNIQUE(repo_full_name)
);

CREATE TABLE stars (
 star_id      INTEGER PRIMARY KEY AUTOINCREMENT
,star_repo_id INTEGER
,star_user_id INTEGER

,UNIQUE(star_repo_id, star_user_id)
);

CREATE INDEX ix_star_user ON stars (star_user_id);

CREATE TABLE keys (
 key_id      INTEGER PRIMARY KEY AUTOINCREMENT
,key_repo_id INTEGER
,key_public  BLOB
,key_private BLOB

,UNIQUE(key_repo_id)
);

CREATE TABLE builds (
 build_id        INTEGER PRIMARY KEY AUTOINCREMENT
,build_repo_id   INTEGER
,build_number    INTEGER
,build_event     TEXT
,build_status    TEXT
,build_enqueued  INTEGER
,build_created   INTEGER
,build_started   INTEGER
,build_finished  INTEGER
,build_commit    TEXT
,build_branch    TEXT
,build_ref       TEXT
,build_refspec   TEXT
,build_remote    TEXT
,build_title     TEXT
,build_message   TEXT
,build_timestamp INTEGER
,build_author    TEXT
,build_avatar    TEXT
,build_email     TEXT
,build_link      TEXT

,UNIQUE(build_number, build_repo_id)
);

CREATE INDEX ix_build_repo   ON builds (build_repo_id);
CREATE INDEX ix_build_author ON builds (build_author);

CREATE TABLE jobs (
 job_id          INTEGER PRIMARY KEY AUTOINCREMENT
,job_node_id     INTEGER
,job_build_id    INTEGER
,job_number      INTEGER
,job_status      TEXT
,job_exit_code   INTEGER
,job_enqueued    INTEGER
,job_started     INTEGER
,job_finished    INTEGER
,job_environment TEXT

,UNIQUE(job_build_id, job_number)
);

CREATE INDEX ix_job_build ON jobs (job_build_id);
CREATE INDEX ix_job_node  ON jobs (job_node_id);

CREATE TABLE IF NOT EXISTS logs (
 log_id     INTEGER PRIMARY KEY AUTOINCREMENT
,log_job_id INTEGER
,log_data   BLOB

,UNIQUE(log_job_id)
);

CREATE TABLE IF NOT EXISTS nodes (
 node_id     INTEGER PRIMARY KEY AUTOINCREMENT
,node_addr   TEXT
,node_arch   TEXT
,node_cert   BLOB
,node_key    BLOB
,node_ca     BLOB
);

INSERT INTO nodes VALUES(null, 'unix:///var/run/docker.sock', 'linux_amd64', '', '', '');
INSERT INTO nodes VALUES(null, 'unix:///var/run/docker.sock', 'linux_amd64', '', '', '');

-- +migrate Down

DROP TABLE nodes;
DROP TABLE logs;
DROP TABLE jobs;
DROP TABLE builds;
DROP TABLE keys;
DROP TABLE stars;
DROP TABLE repos;
DROP TABLE users;