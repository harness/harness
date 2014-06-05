package database

import (
	"database/sql"
	"log"
)

// statements to setup our database
var stmts = []string{`
	CREATE TABLE IF NOT EXISTS users (
		 user_id           INTEGER PRIMARY KEY AUTOINCREMENT
		,user_parent_id    INTEGER
		,user_remote       VARCHAR(255)
		,user_login        VARCHAR(255)
		,user_access       VARCHAR(255)
		,user_secret       VARCHAR(255)
		,user_name         VARCHAR(255)
		,user_email        VARCHAR(255)
		,user_gravatar     VARCHAR(255)
		,user_token        VARCHAR(255)
		,user_admin        BOOLEAN
		,user_active       BOOLEAN
		,user_created      INTEGER
		,user_updated      INTEGER
		,user_synced       INTEGER
		,UNIQUE(user_token)
		,UNIQUE(user_remote, user_login)
	);`, `
	CREATE TABLE IF NOT EXISTS perms (
		 perm_id           INTEGER PRIMARY KEY AUTOINCREMENT
		,user_id           INTEGER
		,repo_id           INTEGER
		,perm_read         BOOLEAN
		,perm_write        BOOLEAN
		,perm_admin        BOOLEAN
		,perm_created      INTEGER
		,perm_updated      INTEGER
		,UNIQUE (repo_id, user_id)
	);`, `
	CREATE TABLE IF NOT EXISTS repos (
		 repo_id           INTEGER PRIMARY KEY AUTOINCREMENT
		,user_id           INTEGER
		,repo_remote       VARCHAR(255)
		,repo_host         VARCHAR(255)
		,repo_owner        VARCHAR(255)
		,repo_name         VARCHAR(255)
		,repo_url          VARCHAR(1024)
		,repo_clone_url    VARCHAR(255)
		,repo_git_url      VARCHAR(255)
		,repo_ssh_url      VARCHAR(255)
		,repo_active       BOOLEAN
		,repo_private      BOOLEAN
		,repo_privileged   BOOLEAN
		,repo_post_commit  BOOLEAN
		,repo_pull_request BOOLEAN
		,repo_public_key   VARCHAR(4000)
		,repo_private_key  VARCHAR(4000)
		,repo_params       VARCHAR(4000)
		,repo_timeout      INTEGER
		,repo_created      INTEGER
		,repo_updated      INTEGER
		,UNIQUE(repo_remote, repo_owner, repo_name)
	);`, `
	CREATE TABLE IF NOT EXISTS commits (
		 commit_id         INTEGER PRIMARY KEY AUTOINCREMENT
		,repo_id           INTEGER
		,commit_status     VARCHAR(255)
		,commit_started    INTEGER
		,commit_finished   INTEGER
		,commit_duration   INTEGER
		,commit_sha        VARCHAR(255)
		,commit_branch     VARCHAR(255)
		,commit_pr         VARCHAR(255)
		,commit_author     VARCHAR(255)
		,commit_gravatar   VARCHAR(255)
		,commit_timestamp  VARCHAR(255)
		,commit_message    VARCHAR(255)
		,commit_yaml       VARCHAR(4000)
		,commit_created    INTEGER
		,commit_updated    INTEGER
		,UNIQUE(commit_sha, commit_branch, repo_id)
	);`, `
	CREATE TABLE IF NOT EXISTS builds (
		 build_id          INTEGER PRIMARY KEY AUTOINCREMENT
		,commit_id         INTEGER
		,build_number      INTEGER
		,build_matrix      VARCHAR(255)
		,build_status      VARCHAR(255)
		,build_console     BLOB
		,build_started     INTEGER
		,build_finished    INTEGER
		,build_duration    INTEGER
		,build_created     INTEGER
		,build_updated     INTEGER
		,UNIQUE(commit_id, build_number)
	);`,
	`CREATE INDEX IF NOT EXISTS builds_commit_id ON builds (commit_id)`,
}

func Load(db *sql.DB) {
	// execute all setup commands
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			// exit on failure since this should never happen
			log.Fatalf("Error generating database schema. %s\n%s", err, stmt)
		}
	}
}
