-- +migrate Up

CREATE TABLE IF NOT EXISTS users (
 user_id     INTEGER PRIMARY KEY AUTO_INCREMENT
,user_login  VARCHAR(255)
,user_token  VARCHAR(500)
,user_secret VARCHAR(500)
,user_expiry INTEGER
,user_email  VARCHAR(500)
,user_avatar VARCHAR(500)
,user_active BOOLEAN
,user_admin  BOOLEAN
,user_hash   VARCHAR(500)

,UNIQUE(user_login)
);

CREATE TABLE IF NOT EXISTS repos (
 repo_id            INTEGER PRIMARY KEY AUTO_INCREMENT
,repo_user_id       INTEGER
,repo_owner         VARCHAR(255)
,repo_name          VARCHAR(255)
,repo_full_name     VARCHAR(255)
,repo_avatar        VARCHAR(500)
,repo_link          VARCHAR(1000)
,repo_clone         VARCHAR(1000)
,repo_branch        VARCHAR(500)
,repo_timeout       INTEGER
,repo_private       BOOLEAN
,repo_trusted       BOOLEAN
,repo_allow_pr      BOOLEAN
,repo_allow_push    BOOLEAN
,repo_allow_deploys BOOLEAN
,repo_allow_tags    BOOLEAN
,repo_hash          VARCHAR(500)

,UNIQUE(repo_full_name)
);

CREATE TABLE IF NOT EXISTS `keys` (
 key_id      INTEGER PRIMARY KEY AUTO_INCREMENT
,key_repo_id INTEGER
,key_public  MEDIUMBLOB
,key_private MEDIUMBLOB

,UNIQUE(key_repo_id)
);

CREATE TABLE IF NOT EXISTS builds (
 build_id        INTEGER PRIMARY KEY AUTO_INCREMENT
,build_repo_id   INTEGER
,build_number    INTEGER
,build_event     VARCHAR(500)
,build_status    VARCHAR(500)
,build_enqueued  INTEGER
,build_created   INTEGER
,build_started   INTEGER
,build_finished  INTEGER
,build_commit    VARCHAR(500)
,build_branch    VARCHAR(500)
,build_ref       VARCHAR(500)
,build_refspec   VARCHAR(1000)
,build_remote    VARCHAR(500)
,build_title     VARCHAR(1000)
,build_message   VARCHAR(2000)
,build_timestamp INTEGER
,build_author    VARCHAR(500)
,build_avatar    VARCHAR(1000)
,build_email     VARCHAR(500)
,build_link      VARCHAR(1000)

,UNIQUE(build_number, build_repo_id)
);

CREATE INDEX ix_build_repo ON builds (build_repo_id);

CREATE TABLE IF NOT EXISTS jobs (
 job_id          INTEGER PRIMARY KEY AUTO_INCREMENT
,job_node_id     INTEGER
,job_build_id    INTEGER
,job_number      INTEGER
,job_status      VARCHAR(500)
,job_exit_code   INTEGER
,job_started     INTEGER
,job_enqueued    INTEGER
,job_finished    INTEGER
,job_environment VARCHAR(2000)

,UNIQUE(job_build_id, job_number)
);

CREATE INDEX ix_job_build ON jobs (job_build_id);
CREATE INDEX ix_job_node  ON jobs (job_node_id);

CREATE TABLE IF NOT EXISTS logs (
 log_id     INTEGER PRIMARY KEY AUTO_INCREMENT
,log_job_id INTEGER
,log_data   MEDIUMBLOB

,UNIQUE(log_job_id)
);

CREATE TABLE IF NOT EXISTS nodes (
 node_id     INTEGER PRIMARY KEY AUTO_INCREMENT
,node_addr   VARCHAR(1024)
,node_arch   VARCHAR(50)
,node_cert   MEDIUMBLOB
,node_key    MEDIUMBLOB
,node_ca     MEDIUMBLOB
);


INSERT INTO nodes VALUES(null, 'unix:///var/run/docker.sock', 'linux_amd64', '', '', '');
INSERT INTO nodes VALUES(null, 'unix:///var/run/docker.sock', 'linux_amd64', '', '', '');

-- +migrate Down

DROP TABLE nodes;
DROP TABLE logs;
DROP TABLE jobs;
DROP TABLE builds;
DROP TABLE `keys`;
DROP TABLE repos;
DROP TABLE users;
