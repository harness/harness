package migrate

import (
	"github.com/drone/drone/Godeps/_workspace/src/github.com/BurntSushi/migration"
)

// Setup is the database migration function that
// will setup the initial SQL database structure.
func Setup(tx migration.LimitedTx) error {
	var stmts = []string{
		userTable,
		starTable,
		repoTable,
		repoKeyTable,
		repoKeyIndex,
		repoParamTable,
		repoParamsIndex,
		repoUserIndex,
		commitTable,
		commitRepoIndex,
		tokenTable,
		buildTable,
		buildCommitIndex,
		statusTable,
		statusCommitIndex,
		blobTable,
		agentTable,
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
	,user_login        VARCHAR(255)
	,user_token        VARCHAR(255)
	,user_secret       VARCHAR(255)
	,user_name         VARCHAR(255)
	,user_email        VARCHAR(255)
	,user_gravatar     VARCHAR(255)
	,user_admin        BOOLEAN
	,user_active       BOOLEAN
	,user_created      INTEGER
	,user_updated      INTEGER
	,UNIQUE(user_login)
);
`

var repoTable = `
CREATE TABLE IF NOT EXISTS repos (
	 repo_id           INTEGER PRIMARY KEY AUTOINCREMENT
	,user_id           INTEGER
	,repo_owner        VARCHAR(255)
	,repo_name         VARCHAR(255)
	,repo_slug         VARCHAR(1024)
	,repo_token        VARCHAR(255)
	,repo_lang         VARCHAR(255)
	,repo_branch       VARCHAR(255)
	,repo_private      BOOLEAN
	,repo_trusted      BOOLEAN
	,repo_self         VARCHAR(1024)
	,repo_link         VARCHAR(1024)
	,repo_clone        VARCHAR(1024)
	,repo_push         BOOLEAN
	,repo_pull         BOOLEAN
	,repo_public_key   BLOB
	,repo_private_key  BLOB
	,repo_params       BLOB
	,repo_timeout      INTEGER
	,repo_created      INTEGER
	,repo_updated      INTEGER
	,UNIQUE(repo_owner, repo_name)
	,UNIQUE(repo_slug)
);
`

var repoUserIndex = `
CREATE INDEX repos_user_idx ON repos (user_id);
`

var repoKeyTable = `
CREATE TABLE IF NOT EXISTS repo_keys (
	 keys_id       INTEGER PRIMARY KEY AUTOINCREMENT
	,repo_id       INTEGER
	,keys_public   BLOB
	,keys_private  BLOB
	,UNIQUE(repo_id)
);
`

var repoKeyIndex = `
CREATE INDEX keys_repo_idx ON repo_keys (repo_id);
`

var repoParamTable = `
CREATE TABLE IF NOT EXISTS repo_params (
	 param_id      INTEGER PRIMARY KEY AUTOINCREMENT
	,repo_id       INTEGER
	,param_map    BLOB
	,UNIQUE(repo_id)
);
`

var repoParamsIndex = `
CREATE INDEX params_repo_idx ON repo_params (repo_id);
`

var starTable = `
CREATE TABLE IF NOT EXISTS stars (
	 star_id  INTEGER PRIMARY KEY AUTOINCREMENT
	,user_id  INTEGER
	,repo_id  INTEGER
	,UNIQUE (repo_id, user_id)
);
`

var commitTable = `
CREATE TABLE IF NOT EXISTS commits (
	 commit_id             INTEGER PRIMARY KEY AUTOINCREMENT
	,repo_id               INTEGER
	,commit_seq            INTEGER
	,commit_state          VARCHAR(255)
	,commit_started        INTEGER
	,commit_finished       INTEGER
	,commit_sha            VARCHAR(255)
	,commit_ref            VARCHAR(255)
	,commit_branch         VARCHAR(255)
	,commit_pr             VARCHAR(255)
	,commit_author         VARCHAR(255)
	,commit_gravatar       VARCHAR(255)
	,commit_timestamp      VARCHAR(255)
	,commit_message        VARCHAR(1000)
	,commit_source_remote  VARCHAR(255)
	,commit_source_branch  VARCHAR(255)
	,commit_source_sha     VARCHAR(255)
	,commit_created        INTEGER
	,commit_updated        INTEGER
	,UNIQUE(repo_id, commit_seq)
	,UNIQUE(repo_id, commit_sha, commit_ref)
);
`

var commitRepoIndex = `
CREATE INDEX commits_repo_idx ON commits (repo_id);
`

var tokenTable = `
CREATE TABLE IF NOT EXISTS tokens (
	 token_id         INTEGER PRIMARY KEY AUTOINCREMENT
	,user_id          INTEGER
	,token_kind       VARCHAR(255)
	,token_label      VARCHAR(255)
	,token_expiry     INTEGER
	,token_issued     INTEGER
	,UNIQUE(user_id, token_label)
);
`

var tokenUserIndex = `
CREATE INDEX tokens_user_idx ON tokens (user_id);
`

var buildTable = `
CREATE TABLE IF NOT EXISTS builds (
	 build_id          INTEGER PRIMARY KEY AUTOINCREMENT
	,commit_id         INTEGER
	,build_seq         INTEGER
	,build_state       VARCHAR(255)
	,build_exit        INTEGER
	,build_duration    INTEGER
	,build_started     INTEGER
	,build_finished    INTEGER
	,build_created     INTEGER
	,build_updated     INTEGER
	,build_env         BLOB
	,UNIQUE(commit_id, build_seq)
);
`

var buildCommitIndex = `
CREATE INDEX builds_commit_idx ON builds (commit_id);
`

var statusTable = `
CREATE TABLE IF NOT EXISTS status (
	 status_id          INTEGER PRIMARY KEY AUTOINCREMENT
	,commit_id          INTEGER
	,status_state       VARCHAR(255)
	,status_desc        VARCHAR(2000)
	,status_link        VARCHAR(2000)
	,status_context     INTEGER
	,status_attachment  BOOL
	,UNIQUE(commit_id, status_context)
);
`

var statusCommitIndex = `
CREATE INDEX status_commit_idx ON status (commit_id);
`

var blobTable = `
CREATE TABLE IF NOT EXISTS blobs (
	 blob_id      INTEGER PRIMARY KEY AUTOINCREMENT
	,blob_path    VARCHAR(255)
	,blob_data    BLOB
	,UNIQUE(blob_path)
);
`

var agentTable = `
CREATE TABLE IF NOT EXISTS agents (
	 agent_id           INTEGER PRIMARY KEY AUTOINCREMENT
	,commit_id          INTEGER
	,agent_addr         VARCHAR(2000)
	,UNIQUE(commit_id)
);
`
