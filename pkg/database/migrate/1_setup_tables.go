package migrate

type rev1st struct{}

var SetupTables = &rev1st{}

func (r *rev1st) Revision() int64 {
	return 1
}

func (r *rev1st) Up(mg *MigrationDriver) error {
	t := mg.T
	if _, err := mg.CreateTable("users", []string{
		t.Integer("id", PRIMARYKEY, AUTOINCREMENT),
		t.String("email", UNIQUE),
		t.String("password"),
		t.String("token", UNIQUE),
		t.String("name"),
		t.String("gravatar"),
		t.Timestamp("created"),
		t.Timestamp("updated"),
		t.Bool("admin"),
		t.String("github_login"),
		t.String("github_token"),
		t.String("bitbucket_login"),
		t.String("bitbucket_token"),
		t.String("bitbucket_secret"),
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("teams", []string{
		t.Integer("id", PRIMARYKEY, AUTOINCREMENT),
		t.String("slug", UNIQUE),
		t.String("name"),
		t.String("email"),
		t.String("gravatar"),
		t.Timestamp("created"),
		t.Timestamp("updated"),
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("members", []string{
		t.Integer("id", PRIMARYKEY, AUTOINCREMENT),
		t.Integer("team_id"),
		t.Integer("user_id"),
		t.String("role"),
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("repos", []string{
		t.Integer("id", PRIMARYKEY, AUTOINCREMENT),
		t.String("slug", UNIQUE),
		t.String("host"),
		t.String("owner"),
		t.String("name"),
		t.Bool("private"),
		t.Bool("disabled"),
		t.Bool("disabled_pr"),
		t.Bool("priveleged"),
		t.Integer("timeout"),
		t.Varchar("scm", 25),
		t.Varchar("url", 1024),
		t.String("username"),
		t.String("password"),
		t.Varchar("public_key", 1024),
		t.Varchar("private_key", 1024),
		t.Blob("params"),
		t.Timestamp("created"),
		t.Timestamp("updated"),
		t.Integer("user_id"),
		t.Integer("team_id"),
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("commits", []string{
		t.Integer("id", PRIMARYKEY, AUTOINCREMENT),
		t.Integer("repo_id"),
		t.String("status"),
		t.Timestamp("started"),
		t.Timestamp("finished"),
		t.Integer("duration"),
		t.Integer("attempts"),
		t.String("hash"),
		t.String("branch"),
		t.String("pull_request"),
		t.String("author"),
		t.String("gravatar"),
		t.String("timestamp"),
		t.String("message"),
		t.Timestamp("created"),
		t.Timestamp("updated"),
	}); err != nil {
		return err
	}

	if _, err := mg.CreateTable("builds", []string{
		t.Integer("id", PRIMARYKEY, AUTOINCREMENT),
		t.Integer("commit_id"),
		t.String("slug"),
		t.String("status"),
		t.Timestamp("started"),
		t.Timestamp("finished"),
		t.Integer("duration"),
		t.Timestamp("created"),
		t.Timestamp("updated"),
		t.Text("stdout"),
	}); err != nil {
		return err
	}

	_, err := mg.CreateTable("settings", []string{
		t.Integer("id", PRIMARYKEY, AUTOINCREMENT),
		t.String("github_key"),
		t.String("github_secret"),
		t.String("bitbucket_key"),
		t.String("bitbucket_secret"),
		t.Varchar("smtp_server", 1024),
		t.Varchar("smtp_port", 5),
		t.Varchar("smtp_address", 1024),
		t.Varchar("smtp_username", 1024),
		t.Varchar("smtp_password", 1024),
		t.Varchar("hostname", 1024),
		t.Varchar("scheme", 5),
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
