package migrate

type rev1st struct{}

var SetupTables = &rev1st{}

func (r *rev1st) Revision() int64 {
	return 1
}

func (r *rev1st) Up(mg *MigrationDriver) error {
	if _, err := mg.CreateTable("users", []string{
		"id                INTEGER PRIMARY KEY AUTOINCREMENT",
		"email             VARCHAR(255) UNIQUE",
		"password          VARCHAR(255)",
		"token             VARCHAR(255) UNIQUE",
		"name              VARCHAR(255)",
		"gravatar          VARCHAR(255)",
		"created           TIMESTAMP",
		"updated           TIMESTAMP",
		"admin             BOOLEAN",
		"github_login      VARCHAR(255)",
		"github_token      VARCHAR(255)",
		"bitbucket_login   VARCHAR(255)",
		"bitbucket_token   VARCHAR(255)",
		"bitbucket_secret  VARCHAR(255)",
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("teams", []string{
		"id       INTEGER PRIMARY KEY AUTOINCREMENT",
		"slug     VARCHAR(255) UNIQUE",
		"name     VARCHAR(255)",
		"email    VARCHAR(255)",
		"gravatar VARCHAR(255)",
		"created  TIMESTAMP",
		"updated  TIMESTAMP",
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("members", []string{
		"id      INTEGER PRIMARY KEY AUTOINCREMENT",
		"team_id INTEGER",
		"user_id INTEGER",
		"role    INTEGER",
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("repos", []string{
		"id          INTEGER PRIMARY KEY AUTOINCREMENT",
		"slug        VARCHAR(1024) UNIQUE",
		"host        VARCHAR(255)",
		"owner       VARCHAR(255)",
		"name        VARCHAR(255)",
		"private     BOOLEAN",
		"disabled    BOOLEAN",
		"disabled_pr BOOLEAN",
		"priveleged  BOOLEAN",
		"timeout     INTEGER",
		"scm         VARCHAR(25)",
		"url         VARCHAR(1024)",
		"username    VARCHAR(255)",
		"password    VARCHAR(255)",
		"public_key  VARCHAR(1024)",
		"private_key VARCHAR(1024)",
		"params      VARCHAR(2000)",
		"created     TIMESTAMP",
		"updated     TIMESTAMP",
		"user_id     INTEGER",
		"team_id     INTEGER",
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("commits", []string{
		"id           INTEGER PRIMARY KEY AUTOINCREMENT",
		"repo_id      INTEGER",
		"status       VARCHAR(255)",
		"started      TIMESTAMP",
		"finished     TIMESTAMP",
		"duration     INTEGER",
		"attempts     INTEGER",
		"hash         VARCHAR(255)",
		"branch       VARCHAR(255)",
		"pull_request VARCHAR(255)",
		"author       VARCHAR(255)",
		"gravatar     VARCHAR(255)",
		"timestamp    VARCHAR(255)",
		"message      VARCHAR(255)",
		"created      TIMESTAMP",
		"updated      TIMESTAMP",
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("builds", []string{
		"id        INTEGER PRIMARY KEY AUTOINCREMENT",
		"commit_id INTEGER",
		"slug      VARCHAR(255)",
		"status    VARCHAR(255)",
		"started   TIMESTAMP",
		"finished  TIMESTAMP",
		"duration  INTEGER",
		"created   TIMESTAMP",
		"updated   TIMESTAMP",
		"stdout    BLOB",
	}); err != nil {
		return err
	}

	_, err := mg.CreateTable("settings", []string{
		"id               INTEGER PRIMARY KEY",
		"github_key       VARCHAR(255)",
		"github_secret    VARCHAR(255)",
		"bitbucket_key    VARCHAR(255)",
		"bitbucket_secret VARCHAR(255)",
		"smtp_server      VARCHAR(1024)",
		"smtp_port        VARCHAR(5)",
		"smtp_address     VARCHAR(1024)",
		"smtp_username    VARCHAR(1024)",
		"smtp_password    VARCHAR(1024)",
		"hostname         VARCHAR(1024)",
		"scheme           VARCHAR(5)",
	})
	return err
}

func (r *rev1st) Down(mg *MigrationDriver) error {
	if _, err := mg.DropTable("settings"); err != nil {
		return err
	}
	if _, err := mg.DropTable("builds"); err != nil {
		return err
	}
	if _, err := mg.DropTable("commits"); err != nil {
		return err
	}
	if _, err := mg.DropTable("repos"); err != nil {
		return err
	}
	if _, err := mg.DropTable("members"); err != nil {
		return err
	}
	if _, err := mg.DropTable("teams"); err != nil {
		return err
	}
	_, err := mg.DropTable("users")
	return err
}
