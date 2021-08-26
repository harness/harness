package mysql

import (
	"database/sql"
)

var migrations = []struct {
	name string
	stmt string
}{
	{
		name: "create-table-users",
		stmt: createTableUsers,
	},
	{
		name: "create-table-repos",
		stmt: createTableRepos,
	},
	{
		name: "alter-table-repos-add-column-no-fork",
		stmt: alterTableReposAddColumnNoFork,
	},
	{
		name: "alter-table-repos-add-column-no-pulls",
		stmt: alterTableReposAddColumnNoPulls,
	},
	{
		name: "alter-table-repos-add-column-cancel-pulls",
		stmt: alterTableReposAddColumnCancelPulls,
	},
	{
		name: "alter-table-repos-add-column-cancel-push",
		stmt: alterTableReposAddColumnCancelPush,
	},
	{
		name: "alter-table-repos-add-column-throttle",
		stmt: alterTableReposAddColumnThrottle,
	},
	{
		name: "alter-table-repos-add-column-cancel-running",
		stmt: alterTableReposAddColumnCancelRunning,
	},
	{
		name: "create-table-perms",
		stmt: createTablePerms,
	},
	{
		name: "create-index-perms-user",
		stmt: createIndexPermsUser,
	},
	{
		name: "create-index-perms-repo",
		stmt: createIndexPermsRepo,
	},
	{
		name: "create-table-builds",
		stmt: createTableBuilds,
	},
	{
		name: "create-index-builds-repo",
		stmt: createIndexBuildsRepo,
	},
	{
		name: "create-index-builds-author",
		stmt: createIndexBuildsAuthor,
	},
	{
		name: "create-index-builds-sender",
		stmt: createIndexBuildsSender,
	},
	{
		name: "create-index-builds-ref",
		stmt: createIndexBuildsRef,
	},
	{
		name: "alter-table-builds-add-column-debug",
		stmt: alterTableBuildsAddColumnDebug,
	},
	{
		name: "create-table-stages",
		stmt: createTableStages,
	},
	{
		name: "create-index-stages-build",
		stmt: createIndexStagesBuild,
	},
	{
		name: "create-table-unfinished",
		stmt: createTableUnfinished,
	},
	{
		name: "create-trigger-stage-insert",
		stmt: createTriggerStageInsert,
	},
	{
		name: "create-trigger-stage-update",
		stmt: createTriggerStageUpdate,
	},
	{
		name: "alter-table-stages-add-column-limit-repos",
		stmt: alterTableStagesAddColumnLimitRepos,
	},
	{
		name: "create-table-steps",
		stmt: createTableSteps,
	},
	{
		name: "create-index-steps-stage",
		stmt: createIndexStepsStage,
	},
	{
		name: "create-table-logs",
		stmt: createTableLogs,
	},
	{
		name: "create-table-cron",
		stmt: createTableCron,
	},
	{
		name: "create-index-cron-repo",
		stmt: createIndexCronRepo,
	},
	{
		name: "create-index-cron-next",
		stmt: createIndexCronNext,
	},
	{
		name: "create-table-secrets",
		stmt: createTableSecrets,
	},
	{
		name: "create-index-secrets-repo",
		stmt: createIndexSecretsRepo,
	},
	{
		name: "create-index-secrets-repo-name",
		stmt: createIndexSecretsRepoName,
	},
	{
		name: "create-table-nodes",
		stmt: createTableNodes,
	},
	{
		name: "alter-table-builds-add-column-cron",
		stmt: alterTableBuildsAddColumnCron,
	},
	{
		name: "create-table-org-secrets",
		stmt: createTableOrgSecrets,
	},
	{
		name: "alter-table-builds-add-column-deploy-id",
		stmt: alterTableBuildsAddColumnDeployId,
	},
	{
		name: "create-table-latest",
		stmt: createTableLatest,
	},
	{
		name: "create-index-latest-repo",
		stmt: createIndexLatestRepo,
	},
	{
		name: "create-table-template",
		stmt: createTableTemplate,
	},
	{
		name: "create-index-template-namespace",
		stmt: createIndexTemplateNamespace,
	},
	{
		name: "alter-table-steps-add-column-step-depends-on",
		stmt: alterTableStepsAddColumnStepDependsOn,
	},
	{
		name: "alter-table-steps-add-column-step-image",
		stmt: alterTableStepsAddColumnStepImage,
	},
	{
		name: "alter-table-steps-add-column-step-detached",
		stmt: alterTableStepsAddColumnStepDetached,
	},
}

// Migrate performs the database migration. If the migration fails
// and error is returned.
func Migrate(db *sql.DB) error {
	if err := createTable(db); err != nil {
		return err
	}
	completed, err := selectCompleted(db)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	for _, migration := range migrations {
		if _, ok := completed[migration.name]; ok {

			continue
		}

		if _, err := db.Exec(migration.stmt); err != nil {
			return err
		}
		if err := insertMigration(db, migration.name); err != nil {
			return err
		}

	}
	return nil
}

func createTable(db *sql.DB) error {
	_, err := db.Exec(migrationTableCreate)
	return err
}

func insertMigration(db *sql.DB, name string) error {
	_, err := db.Exec(migrationInsert, name)
	return err
}

func selectCompleted(db *sql.DB) (map[string]struct{}, error) {
	migrations := map[string]struct{}{}
	rows, err := db.Query(migrationSelect)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		migrations[name] = struct{}{}
	}
	return migrations, nil
}

//
// migration table ddl and sql
//

var migrationTableCreate = `
CREATE TABLE IF NOT EXISTS migrations (
 name VARCHAR(255)
,UNIQUE(name)
)
`

var migrationInsert = `
INSERT INTO migrations (name) VALUES (?)
`

var migrationSelect = `
SELECT name FROM migrations
`

//
// 001_create_table_user.sql
//

var createTableUsers = `
CREATE TABLE IF NOT EXISTS users (
 user_id            INTEGER PRIMARY KEY AUTO_INCREMENT
,user_login         VARCHAR(250)
,user_email         VARCHAR(500)
,user_admin         BOOLEAN
,user_machine       BOOLEAN
,user_active        BOOLEAN
,user_avatar        VARCHAR(2000)
,user_syncing       BOOLEAN
,user_synced        INTEGER
,user_created       INTEGER
,user_updated       INTEGER
,user_last_login    INTEGER
,user_oauth_token   BLOB
,user_oauth_refresh BLOB
,user_oauth_expiry  INTEGER
,user_hash          VARCHAR(500)
,UNIQUE(user_login)
,UNIQUE(user_hash)
);
`

//
// 002_create_table_repos.sql
//

var createTableRepos = `
CREATE TABLE IF NOT EXISTS repos (
 repo_id                    INTEGER PRIMARY KEY AUTO_INCREMENT
,repo_uid                   VARCHAR(250)
,repo_user_id               INTEGER
,repo_namespace             VARCHAR(250)
,repo_name                  VARCHAR(250)
,repo_slug                  VARCHAR(250)
,repo_scm                   VARCHAR(50)
,repo_clone_url             VARCHAR(2000)
,repo_ssh_url               VARCHAR(2000)
,repo_html_url              VARCHAR(2000)
,repo_active                BOOLEAN
,repo_private               BOOLEAN
,repo_visibility            VARCHAR(50)
,repo_branch                VARCHAR(250)
,repo_counter               INTEGER
,repo_config                VARCHAR(500)
,repo_timeout               INTEGER
,repo_trusted               BOOLEAN
,repo_protected             BOOLEAN
,repo_synced                INTEGER
,repo_created               INTEGER
,repo_updated               INTEGER
,repo_version               INTEGER
,repo_signer                VARCHAR(50)
,repo_secret                VARCHAR(50)
,UNIQUE(repo_slug)
,UNIQUE(repo_uid)
);
`

var alterTableReposAddColumnNoFork = `
ALTER TABLE repos ADD COLUMN repo_no_forks BOOLEAN NOT NULL DEFAULT false;
`

var alterTableReposAddColumnNoPulls = `
ALTER TABLE repos ADD COLUMN repo_no_pulls BOOLEAN NOT NULL DEFAULT false;
`

var alterTableReposAddColumnCancelPulls = `
ALTER TABLE repos ADD COLUMN repo_cancel_pulls BOOLEAN NOT NULL DEFAULT false;
`

var alterTableReposAddColumnCancelPush = `
ALTER TABLE repos ADD COLUMN repo_cancel_push BOOLEAN NOT NULL DEFAULT false;
`

var alterTableReposAddColumnThrottle = `
ALTER TABLE repos ADD COLUMN repo_throttle INTEGER NOT NULL DEFAULT 0;
`

var alterTableReposAddColumnCancelRunning = `
ALTER TABLE repos ADD COLUMN repo_cancel_running BOOLEAN NOT NULL DEFAULT false;
`

//
// 003_create_table_perms.sql
//

var createTablePerms = `
CREATE TABLE IF NOT EXISTS perms (
 perm_user_id  INTEGER
,perm_repo_uid VARCHAR(250)
,perm_read     BOOLEAN
,perm_write    BOOLEAN
,perm_admin    BOOLEAN
,perm_synced   INTEGER
,perm_created  INTEGER
,perm_updated  INTEGER
,PRIMARY KEY(perm_user_id, perm_repo_uid)
);
`

var createIndexPermsUser = `
CREATE INDEX ix_perms_user ON perms (perm_user_id);
`

var createIndexPermsRepo = `
CREATE INDEX ix_perms_repo ON perms (perm_repo_uid);
`

//
// 004_create_table_builds.sql
//

var createTableBuilds = `
CREATE TABLE IF NOT EXISTS builds (
 build_id            INTEGER PRIMARY KEY AUTO_INCREMENT
,build_repo_id       INTEGER
,build_config_id     INTEGER
,build_trigger       VARCHAR(250)
,build_number        INTEGER
,build_parent        INTEGER
,build_status        VARCHAR(50)
,build_error         VARCHAR(500)
,build_event         VARCHAR(50)
,build_action        VARCHAR(50)
,build_link          VARCHAR(1000)
,build_timestamp     INTEGER
,build_title         VARCHAR(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
,build_message       VARCHAR(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
,build_before        VARCHAR(50)
,build_after         VARCHAR(50)
,build_ref           VARCHAR(500)
,build_source_repo   VARCHAR(250)
,build_source        VARCHAR(500)
,build_target        VARCHAR(500)
,build_author        VARCHAR(500)
,build_author_name   VARCHAR(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
,build_author_email  VARCHAR(500)
,build_author_avatar VARCHAR(1000)
,build_sender        VARCHAR(500)
,build_deploy        VARCHAR(500)
,build_params        VARCHAR(2000)
,build_started       INTEGER
,build_finished      INTEGER
,build_created       INTEGER
,build_updated       INTEGER
,build_version       INTEGER
,UNIQUE(build_repo_id, build_number)
);
`

var createIndexBuildsRepo = `
CREATE INDEX ix_build_repo ON builds (build_repo_id);
`

var createIndexBuildsAuthor = `
CREATE INDEX ix_build_author ON builds (build_author);
`

var createIndexBuildsSender = `
CREATE INDEX ix_build_sender ON builds (build_sender);
`

var createIndexBuildsRef = `
CREATE INDEX ix_build_ref ON builds (build_repo_id, build_ref);
`

var alterTableBuildsAddColumnDebug = `
ALTER TABLE builds ADD COLUMN build_debug BOOLEAN NOT NULL DEFAULT false;
`

//
// 005_create_table_stages.sql
//

var createTableStages = `
CREATE TABLE IF NOT EXISTS stages (
 stage_id          INTEGER PRIMARY KEY AUTO_INCREMENT
,stage_repo_id     INTEGER
,stage_build_id    INTEGER
,stage_number      INTEGER
,stage_name        VARCHAR(100)
,stage_kind        VARCHAR(50)
,stage_type        VARCHAR(50)
,stage_status      VARCHAR(50)
,stage_error       VARCHAR(500)
,stage_errignore   BOOLEAN
,stage_exit_code   INTEGER
,stage_limit       INTEGER
,stage_os          VARCHAR(50)
,stage_arch        VARCHAR(50)
,stage_variant     VARCHAR(10)
,stage_kernel      VARCHAR(50)
,stage_machine     VARCHAR(500)
,stage_started     INTEGER
,stage_stopped     INTEGER
,stage_created     INTEGER
,stage_updated     INTEGER
,stage_version     INTEGER
,stage_on_success  BOOLEAN
,stage_on_failure  BOOLEAN
,stage_depends_on  TEXT
,stage_labels      TEXT
,UNIQUE(stage_build_id, stage_number)
);
`

var createIndexStagesBuild = `
CREATE INDEX ix_stages_build ON stages (stage_build_id);
`

var createTableUnfinished = `
CREATE TABLE IF NOT EXISTS stages_unfinished (
stage_id INTEGER PRIMARY KEY
);
`

var createTriggerStageInsert = `
CREATE TRIGGER stage_insert AFTER INSERT ON stages
FOR EACH ROW
BEGIN
   IF NEW.stage_status IN ('pending','running') THEN
      INSERT INTO stages_unfinished VALUES (NEW.stage_id);
   END IF;
END;
`

var createTriggerStageUpdate = `
CREATE TRIGGER stage_update AFTER UPDATE ON stages
FOR EACH ROW
BEGIN
  IF NEW.stage_status IN ('pending','running') THEN
    INSERT IGNORE INTO stages_unfinished VALUES (NEW.stage_id);
  ELSEIF OLD.stage_status IN ('pending','running') THEN
    DELETE FROM stages_unfinished WHERE stage_id = OLD.stage_id;
  END IF;
END;
`

var alterTableStagesAddColumnLimitRepos = `
ALTER TABLE stages ADD COLUMN stage_limit_repo INTEGER NOT NULL DEFAULT 0;
`

//
// 006_create_table_steps.sql
//

var createTableSteps = `
CREATE TABLE IF NOT EXISTS steps (
 step_id          INTEGER PRIMARY KEY AUTO_INCREMENT
,step_stage_id    INTEGER
,step_number      INTEGER
,step_name        VARCHAR(100)
,step_status      VARCHAR(50)
,step_error       VARCHAR(500)
,step_errignore   BOOLEAN
,step_exit_code   INTEGER
,step_started     INTEGER
,step_stopped     INTEGER
,step_version     INTEGER
,UNIQUE(step_stage_id, step_number)
);
`

var createIndexStepsStage = `
CREATE INDEX ix_steps_stage ON steps (step_stage_id);
`

//
// 007_create_table_logs.sql
//

var createTableLogs = `
CREATE TABLE IF NOT EXISTS logs (
 log_id    INTEGER PRIMARY KEY
,log_data  MEDIUMBLOB
);
`

//
// 008_create_table_cron.sql
//

var createTableCron = `
CREATE TABLE IF NOT EXISTS cron (
 cron_id          INTEGER PRIMARY KEY AUTO_INCREMENT
,cron_repo_id     INTEGER
,cron_name        VARCHAR(50)
,cron_expr        VARCHAR(50)
,cron_next        INTEGER
,cron_prev        INTEGER
,cron_event       VARCHAR(50)
,cron_branch      VARCHAR(250)
,cron_target      VARCHAR(250)
,cron_disabled    BOOLEAN
,cron_created     INTEGER
,cron_updated     INTEGER
,cron_version     INTEGER
,UNIQUE(cron_repo_id, cron_name)
,FOREIGN KEY(cron_repo_id) REFERENCES repos(repo_id) ON DELETE CASCADE
);
`

var createIndexCronRepo = `
CREATE INDEX ix_cron_repo ON cron (cron_repo_id);
`

var createIndexCronNext = `
CREATE INDEX ix_cron_next ON cron (cron_next);
`

//
// 009_create_table_secrets.sql
//

var createTableSecrets = `
CREATE TABLE IF NOT EXISTS secrets (
 secret_id                INTEGER PRIMARY KEY AUTO_INCREMENT
,secret_repo_id           INTEGER
,secret_name              VARCHAR(500)
,secret_data              BLOB
,secret_pull_request      BOOLEAN
,secret_pull_request_push BOOLEAN
,UNIQUE(secret_repo_id, secret_name)
,FOREIGN KEY(secret_repo_id) REFERENCES repos(repo_id) ON DELETE CASCADE
);
`

var createIndexSecretsRepo = `
CREATE INDEX ix_secret_repo ON secrets (secret_repo_id);
`

var createIndexSecretsRepoName = `
CREATE INDEX ix_secret_repo_name ON secrets (secret_repo_id, secret_name);
`

//
// 010_create_table_nodes.sql
//

var createTableNodes = `
CREATE TABLE IF NOT EXISTS nodes (
 node_id         INTEGER PRIMARY KEY AUTO_INCREMENT
,node_uid        VARCHAR(500)
,node_provider   VARCHAR(50)
,node_state      VARCHAR(50)
,node_name       VARCHAR(50)
,node_image      VARCHAR(500)
,node_region     VARCHAR(100)
,node_size       VARCHAR(100)
,node_os         VARCHAR(50)
,node_arch       VARCHAR(50)
,node_kernel     VARCHAR(50)
,node_variant    VARCHAR(50)
,node_address    VARCHAR(500)
,node_capacity   INTEGER
,node_filter     VARCHAR(2000)
,node_labels     VARCHAR(2000)
,node_error      VARCHAR(2000)
,node_ca_key     BLOB
,node_ca_cert    BLOB
,node_tls_key    BLOB
,node_tls_cert   BLOB
,node_tls_name   VARCHAR(500)
,node_paused     BOOLEAN
,node_protected  BOOLEAN
,node_created    INTEGER
,node_updated    INTEGER
,node_pulled     INTEGER

,UNIQUE(node_name)
);
`

//
// 011_add_column_builds_cron.sql
//

var alterTableBuildsAddColumnCron = `
ALTER TABLE builds ADD COLUMN build_cron VARCHAR(50) NOT NULL DEFAULT '';
`

//
// 012_create_table_global_secrets.sql
//

var createTableOrgSecrets = `
CREATE TABLE IF NOT EXISTS orgsecrets (
 secret_id                INTEGER PRIMARY KEY AUTO_INCREMENT
,secret_namespace         VARCHAR(50)
,secret_name              VARCHAR(200)
,secret_type              VARCHAR(50)
,secret_data              BLOB
,secret_pull_request      BOOLEAN
,secret_pull_request_push BOOLEAN
,UNIQUE(secret_namespace, secret_name)
);
`

//
// 013_add_column_builds_deploy_id.sql
//

var alterTableBuildsAddColumnDeployId = `
ALTER TABLE builds ADD COLUMN build_deploy_id INTEGER NOT NULL DEFAULT 0;
`

//
// 014_create_table_refs.sql
//

var createTableLatest = `
CREATE TABLE IF NOT EXISTS latest (
 latest_repo_id  INTEGER
,latest_build_id INTEGER
,latest_type     VARCHAR(50)
,latest_name     VARCHAR(500)
,latest_created  INTEGER
,latest_updated  INTEGER
,latest_deleted  INTEGER
,PRIMARY KEY(latest_repo_id, latest_type, latest_name)
);
`

var createIndexLatestRepo = `
CREATE INDEX ix_latest_repo ON latest (latest_repo_id);
`

//
// 015_create_table_templates.sql
//

var createTableTemplate = `
CREATE TABLE IF NOT EXISTS templates (
     template_id      INTEGER PRIMARY KEY AUTO_INCREMENT
    ,template_name    VARCHAR(500)
    ,template_namespace VARCHAR(50)
    ,template_data    BLOB
    ,template_created INTEGER
    ,template_updated INTEGER
    ,UNIQUE(template_name, template_namespace)
    );
`

var createIndexTemplateNamespace = `
CREATE INDEX ix_template_namespace ON templates (template_namespace);
`

//
// 016_add_columns_steps.sql
//

var alterTableStepsAddColumnStepDependsOn = `
ALTER TABLE steps ADD COLUMN step_depends_on TEXT NULL;
`

var alterTableStepsAddColumnStepImage = `
ALTER TABLE steps ADD COLUMN step_image VARCHAR(1000) NOT NULL DEFAULT '';
`

var alterTableStepsAddColumnStepDetached = `
ALTER TABLE steps ADD COLUMN step_detached BOOLEAN NOT NULL DEFAULT FALSE;
`
