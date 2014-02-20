DROP TABLE IF EXISTS builds;
DROP TABLE IF EXISTS commits;
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS settings;

CREATE TABLE users (
	 id       INTEGER PRIMARY KEY AUTOINCREMENT
	,email    VARCHAR(255) UNIQUE
	,password VARCHAR(255)
	,token    VARCHAR(255) UNIQUE
	,name     VARCHAR(255)
	,gravatar VARCHAR(255)
	,created  TIMESTAMP
	,updated  TIMESTAMP
	,admin    BOOLEAN

	,github_login      VARCHAR(255)
	,github_token      VARCHAR(255)

	,bitbucket_login   VARCHAR(255)
	,bitbucket_token   VARCHAR(255)
	,bitbucket_secret  VARCHAR(255)
);

CREATE TABLE teams (
	 id        INTEGER PRIMARY KEY AUTOINCREMENT
	,slug      VARCHAR(255) UNIQUE
	,name      VARCHAR(255) UNIQUE
	,email     VARCHAR(255)
	,gravatar  VARCHAR(255)
	,created   TIMESTAMP
	,updated   TIMESTAMP
);

CREATE TABLE members (
	 id      INTEGER PRIMARY KEY AUTOINCREMENT
	,team_id INTEGER
	,user_id INTEGER
	,role    INTEGER
);

CREATE TABLE repos (
	 id            INTEGER PRIMARY KEY AUTOINCREMENT
	,slug          VARCHAR(1024) UNIQUE
	,host          VARCHAR(255)
	,owner         VARCHAR(255)
	,name          VARCHAR(255)
	,private       BOOLEAN
	,disabled      BOOLEAN
	,disabled_pr   BOOLEAN
	,privileged    BOOLEAN
	,timeout       INTEGER

	,scm         VARCHAR(25)
	,url         VARCHAR(1024)
	,username    VARCHAR(255)
	,password    VARCHAR(255)
	,public_key  VARCHAR(1024)
	,private_key VARCHAR(1024)
	,params      VARCHAR(2000)

	,created     TIMESTAMP
	,updated     TIMESTAMP
	,user_id     INTEGER
	,team_id     INTEGER
);

CREATE TABLE commits (
	 id           INTEGER PRIMARY KEY AUTOINCREMENT
	,repo_id      INTEGER
	,status       VARCHAR(255)
	,started      TIMESTAMP
	,finished     TIMESTAMP
	,duration     INTEGER
	,hash         VARCHAR(255)
	,branch       VARCHAR(255)
	,pull_request VARCHAR(255)
	,author       VARCHAR(255)
	,gravatar     VARCHAR(255)
	,timestamp    VARCHAR(255)
	,message      VARCHAR(255)
	,created      TIMESTAMP
	,updated      TIMESTAMP
);

CREATE TABLE builds (
	 id        INTEGER PRIMARY KEY AUTOINCREMENT
	,commit_id INTEGER
	,slug      VARCHAR(255)
	,status    VARCHAR(255)
	,started   TIMESTAMP
	,finished  TIMESTAMP
	,duration  INTEGER
	,created   TIMESTAMP
	,updated   TIMESTAMP
	,stdout    BLOB
);

CREATE TABLE settings (
     id               INTEGER PRIMARY KEY
    ,github_key       VARCHAR(255)
    ,github_secret    VARCHAR(255)
    ,github_domain    VARCHAR(255)
    ,github_apiurl    VARCHAR(255)
    ,bitbucket_key    VARCHAR(255)
    ,bitbucket_secret VARCHAR(255)
    ,smtp_server      VARCHAR(1024)
    ,smtp_port        VARCHAR(5)
    ,smtp_address     VARCHAR(1024)
    ,smtp_username    VARCHAR(1024)
    ,smtp_password    VARCHAR(1024)
    ,hostname         VARCHAR(1024)
    ,scheme           VARCHAR(5)
    ,open_invitations BOOLEAN
);

CREATE UNIQUE INDEX member_uix       ON members  (team_id, user_id);
CREATE UNIQUE INDEX commits_uix      ON commits  (repo_id, hash, branch);

CREATE INDEX member_team_ix          ON members (team_id);
CREATE INDEX member_user_ix          ON members (user_id);
CREATE INDEX repo_team_ix            ON repos   (team_id);
CREATE INDEX repo_user_ix            ON repos   (user_id);
CREATE INDEX commits_repo_ix         ON commits (repo_id);
CREATE INDEX commits_repo_branch_ix  ON commits (repo_id, branch);
CREATE INDEX builds_commit_ix        ON builds  (commit_id);
CREATE INDEX builds_commit_slug_ix   ON builds  (commit_id, slug);
