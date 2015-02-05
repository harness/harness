package migrate

import (
	"github.com/BurntSushi/migration"
)

// Setup is the database migration function that
// will setup the initial SQL database structure.
func Setup(tx migration.LimitedTx) error {
	var stmts = []string{
		blobTable,
		userTable,
		repoTable,
		permTable,
		commitTable,
	}
	for _, stmt := range stmts {
		_, err := tx.Exec(transform(stmt))
		if err != nil {
			return err
		}
	}
	return nil
}

// Migrate_20142110 is a database migration on Oct-10 2014.
func Migrate_20142110(tx migration.LimitedTx) error {
	var stmts = []string{
		commitRepoIndex, // index the commit table repo_id column
		repoTokenColumn, // add the repo token column
		repoTokenUpdate, // update the repo token column to empty string
	}
	for _, stmt := range stmts {
		_, err := tx.Exec(transform(stmt))
		if err != nil {
			return err
		}
	}
	return nil
}

// Migrate_20142110 is a database migration on Oct-10 2014.
func Migrate_20152701(tx migration.LimitedTx) error {
	var stmts = []string{
		addUserTokenExpires, // index the commit table repo_id column
	}
	for _, stmt := range stmts {
		_, err := tx.Exec(transform(stmt))
		if err != nil {
			return err
		}
	}
	return nil
}

var userTable = `
CREATE TABLE IF NOT EXISTS users (
	 user_id           INTEGER PRIMARY KEY AUTOINCREMENT
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
	,user_syncing      BOOLEAN
	,user_created      INTEGER
	,user_updated      INTEGER
	,user_synced       INTEGER
	,UNIQUE(user_token)
	,UNIQUE(user_remote, user_login)
);
`

var permTable = `
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
);
`

var repoTable = `
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
	,repo_public_key   BLOB
	,repo_private_key  BLOB
	,repo_params       BLOB
	,repo_timeout      INTEGER
	,repo_created      INTEGER
	,repo_updated      INTEGER
	,UNIQUE(repo_host, repo_owner, repo_name)
);
`

var repoTokenColumn = `
ALTER TABLE repos ADD COLUMN repo_token VARCHAR(40)
`

var repoTokenUpdate = `
UPDATE repos SET repo_token = '';
`

var commitTable = `
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
	,commit_yaml       BLOB
	,commit_created    INTEGER
	,commit_updated    INTEGER
	,UNIQUE(commit_sha, commit_branch, repo_id)
);
`

var commitRepoIndex = `
CREATE INDEX commit_repo_id_idx ON commits (repo_id);
`

var blobTable = `
CREATE TABLE IF NOT EXISTS blobs (
	 blob_id      INTEGER PRIMARY KEY AUTOINCREMENT
	,blob_path    VARCHAR(255)
	,blob_data    BLOB
	,UNIQUE(blob_path)
);
`

var addUserTokenExpires = `
ALTER TABLE users ADD COLUMN user_access_expires INTEGER
`
