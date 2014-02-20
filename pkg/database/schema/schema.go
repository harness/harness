package schema

import (
	"database/sql"
)

// SQL statement to create the User Table.
var userTableStmt = `
CREATE TABLE users (
   id                INTEGER PRIMARY KEY AUTOINCREMENT
  ,email             VARCHAR(255) UNIQUE
  ,password          VARCHAR(255)
  ,token             VARCHAR(255) UNIQUE
  ,name              VARCHAR(255)
  ,gravatar          VARCHAR(255)
  ,created           TIMESTAMP
  ,updated           TIMESTAMP
  ,admin             BOOLEAN
  ,github_login      VARCHAR(255)
  ,github_token      VARCHAR(255)
  ,bitbucket_login   VARCHAR(255)
  ,bitbucket_token   VARCHAR(255)
  ,bitbucket_secret  VARCHAR(255)
);
`

// SQL statement to create the Team Table.
var teamTableStmt = `
CREATE TABLE teams (
  id        INTEGER PRIMARY KEY AUTOINCREMENT
  ,slug     VARCHAR(255) UNIQUE
  ,name     VARCHAR(255)
  ,email    VARCHAR(255)
  ,gravatar VARCHAR(255)
  ,created  TIMESTAMP
  ,updated  TIMESTAMP
);
`

// SQL statement to create the Member Table.
var memberTableStmt = `
CREATE TABLE members (
   id      INTEGER PRIMARY KEY AUTOINCREMENT
  ,team_id INTEGER
  ,user_id INTEGER
  ,role    INTEGER
);
`

// SQL statement to create the Repo Table.
var repoTableStmt = `
CREATE TABLE repos (
   id          INTEGER PRIMARY KEY AUTOINCREMENT
  ,slug        VARCHAR(1024) UNIQUE
  ,host        VARCHAR(255)
  ,owner       VARCHAR(255)
  ,name        VARCHAR(255)
  ,private     BOOLEAN
  ,disabled    BOOLEAN
  ,disabled_pr BOOLEAN
  ,priveleged  BOOLEAN
  ,timeout     INTEGER
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
`

// SQL statement to create the Commit Table.
var commitTableStmt = `
CREATE TABLE commits (
   id           INTEGER PRIMARY KEY AUTOINCREMENT
  ,repo_id      INTEGER
  ,status       VARCHAR(255)
  ,started      TIMESTAMP
  ,finished     TIMESTAMP
  ,duration     INTEGER
  ,attempts     INTEGER
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
`

// SQL statement to create the Build Table.
var buildTableStmt = `
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
`

// SQL statement to create the Settings
var settingsTableStmt = `
CREATE TABLE settings (
   id               INTEGER PRIMARY KEY
  ,github_key       VARCHAR(255)
  ,github_secret    VARCHAR(255)
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
`

var memberUniqueIndex = `
CREATE UNIQUE INDEX member_uix ON members (team_id, user_id);
`

var memberTeamIndex = `
CREATE INDEX member_team_ix ON members (team_id);
`

var memberUserIndex = `
CREATE INDEX member_user_ix ON members (user_id);
`

var commitUniqueIndex = `
CREATE UNIQUE INDEX commits_uix ON commits  (repo_id, hash, branch);
`

var commitRepoIndex = `
CREATE INDEX commits_repo_ix ON commits (repo_id);
`

var commitBranchIndex = `
CREATE INDEX commits_repo_ix ON commits (repo_id, branch);
`

var repoTeamIndex = `
CREATE INDEX repo_team_ix ON repos (team_id);
`

var repoUserIndex = `
CREATE INDEX repo_user_ix ON repos (user_id);
`

var buildCommitIndex = `
CREATE INDEX builds_commit_ix ON builds (commit_id);
`

var buildSlugIndex = `
CREATE INDEX builds_commit_slug_ix ON builds (commit_id, slug);
`

// Load will apply the DDL commands to
// the provided database.
func Load(db *sql.DB) error {

	// created tables
	db.Exec(userTableStmt)
	db.Exec(teamTableStmt)
	db.Exec(memberTableStmt)
	db.Exec(repoTableStmt)
	db.Exec(commitTableStmt)
	db.Exec(buildTableStmt)
	db.Exec(settingsTableStmt)

	db.Exec(memberUniqueIndex)
	db.Exec(memberTeamIndex)
	db.Exec(memberUserIndex)
	db.Exec(commitUniqueIndex)
	db.Exec(commitRepoIndex)
	db.Exec(commitBranchIndex)
	db.Exec(repoTeamIndex)
	db.Exec(repoUserIndex)
	db.Exec(buildCommitIndex)
	db.Exec(buildSlugIndex)

	// migrations for backward compatibility
	db.Exec("ALTER TABLE settings ADD COLUMN open_invitations BOOLEAN")
	db.Exec("UPDATE settings SET open_invitations=0 WHERE open_invitations IS NULL")

	return nil
}
