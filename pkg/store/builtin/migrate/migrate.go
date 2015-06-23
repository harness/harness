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
		repoUserIndex,
		buildTable,
		buildRepoIndex,
		buildBranchIndex,
		tokenTable,
		jobTable,
		jobBuildIndex,
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
	,user_email        VARCHAR(255)
	,user_avatar       VARCHAR(255)
	,user_admin        BOOLEAN
	,user_active       BOOLEAN
	,UNIQUE(user_login)
);
`

var repoTable = `
CREATE TABLE IF NOT EXISTS repos (
	 repo_id                 INTEGER PRIMARY KEY AUTOINCREMENT
	,repo_user_id            INTEGER
	,repo_owner              VARCHAR(255)
	,repo_name               VARCHAR(255)
	,repo_full_name          VARCHAR(1024)
	,repo_self               VARCHAR(1024)
	,repo_link               VARCHAR(1024)
	,repo_clone              VARCHAR(1024)
	,repo_branch             VARCHAR(255)
	,repo_private            BOOLEAN
	,repo_trusted            BOOLEAN
	,repo_timeout            INTEGER
	,repo_keys_public	     BLOB
	,repo_keys_private	     BLOB
	,repo_hooks_pull_request BOOLEAN
	,repo_hooks_push         BOOLEAN
	,repo_hooks_tags         BOOLEAN
	,repo_params             BLOB

	,UNIQUE(repo_owner, repo_name)
	,UNIQUE(repo_full_name)
);
`

var repoUserIndex = `
CREATE INDEX repos_user_idx ON repos (repo_user_id);
`

var starTable = `
CREATE TABLE IF NOT EXISTS stars (
	 star_id       INTEGER PRIMARY KEY AUTOINCREMENT
	,star_user_id  INTEGER
	,star_repo_id  INTEGER
	,UNIQUE (star_repo_id, star_user_id)
);
`

var buildTable = `
CREATE TABLE IF NOT EXISTS builds (
	 build_id                             INTEGER PRIMARY KEY AUTOINCREMENT
	,build_repo_id                        INTEGER
	,build_number                         INTEGER
	,build_status                         VARCHAR(512)
	,build_started                        INTEGER
	,build_finished                       INTEGER
	,build_commit_sha                     VARCHAR(512)
	,build_commit_ref                     VARCHAR(512)
	,build_commit_branch                  VARCHAR(512)
	,build_commit_message                 VARCHAR(512)
	,build_commit_timestamp               VARCHAR(512)
	,build_commit_remote                  VARCHAR(512)
	,build_commit_author_login            VARCHAR(512)
	,build_commit_author_email            VARCHAR(512)
	,build_pull_request_number            INTEGER
	,build_pull_request_title             VARCHAR(512)
	,build_pull_request_base_sha          VARCHAR(512)
	,build_pull_request_base_ref          VARCHAR(512)
	,build_pull_request_base_branch       VARCHAR(512)
	,build_pull_request_base_message      VARCHAR(512)
	,build_pull_request_base_timestamp    VARCHAR(512)
	,build_pull_request_base_remote       VARCHAR(512)
	,build_pull_request_base_author_login VARCHAR(512)
	,build_pull_request_base_author_email VARCHAR(512)
	,UNIQUE(build_repo_id, build_number)
);
`

var buildRepoIndex = `
CREATE INDEX build_repo_idx ON builds (build_repo_id);
`

var buildBranchIndex = `
CREATE INDEX build_branch_idx ON builds (build_commit_branch);
`

var tokenTable = `
CREATE TABLE IF NOT EXISTS tokens (
	 token_id         INTEGER PRIMARY KEY AUTOINCREMENT
	,token_user_id    INTEGER
	,token_kind       VARCHAR(255)
	,token_label      VARCHAR(255)
	,token_expiry     INTEGER
	,token_issued     INTEGER
	,UNIQUE(token_user_id, token_label)
);
`

var tokenUserIndex = `
CREATE INDEX tokens_user_idx ON tokens (token_user_id);
`

var jobTable = `
CREATE TABLE IF NOT EXISTS jobs (
	 job_id          INTEGER PRIMARY KEY AUTOINCREMENT
	,job_build_id    INTEGER
	,job_number      INTEGER
	,job_status      VARCHAR(255)
	,job_exit_code   INTEGER
	,job_started     INTEGER
	,job_finished    INTEGER
	,job_environment VARCHAR(2000)
	,UNIQUE(job_build_id, job_number)
);
`

var jobBuildIndex = `
CREATE INDEX ix_job_build_id ON jobs (job_build_id);
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
	,agent_build_id     INTEGER
	,agent_addr         VARCHAR(2000)
	,UNIQUE(agent_build_id)
);
`
