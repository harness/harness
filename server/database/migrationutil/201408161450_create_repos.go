package migrationutil

type Migrate_201408161450 struct{}

var CreateReposTable = &Migrate_201408161450{}

func (m *Migrate_201408161450) Revision() int64 {
	return 201408161450
}

func (m *Migrate_201408161450) Up(mg *MigrationDriver) error {
	t := mg.T

	if _, err := mg.CreateTable("repos", []string{
		t.Pk("repo_id"),
		t.Integer("user_id"),
		t.String("repo_remote"),
		t.String("repo_host"),
		t.String("repo_owner"),
		t.String("repo_name"),
		t.String("repo_url"),
		t.String("repo_clone_url"),
		t.String("repo_git_url"),
		t.String("repo_ssh_url"),
		t.Bool("repo_active"),
		t.Bool("repo_private"),
		t.Bool("repo_privileged"),
		t.Bool("repo_post_commit"),
		t.Bool("repo_pull_request"),
		t.Varchar("repo_public_key", 4000),
		t.Varchar("repo_private_key", 4000),
		t.Varchar("repo_params", 4000),
		t.Integer("repo_timeout"),
		t.Integer("repo_created"),
		t.Integer("repo_updated"),
	}); err != nil {
		return err
	}

	_, err := mg.AddIndex("repos", []string{"repo_host", "repo_owner", "repo_name"}, "unique")
	return err
}

func (m *Migrate_201408161450) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("repos")
	return err
}
