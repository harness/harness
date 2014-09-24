package migrationutil

type Migrate_201408161536 struct{}

var CreateCommitsTable = &Migrate_201408161536{}

func (m *Migrate_201408161536) Revision() int64 {
	return 201408161536
}

func (m *Migrate_201408161536) Up(mg *MigrationDriver) error {
	t := mg.T

	if _, err := mg.CreateTable("commits", []string{
		t.Pk("commit_id"),
		t.Integer("repo_id"),
		t.String("commit_status"),
		t.Integer("commit_started"),
		t.Integer("commit_finished"),
		t.Integer("commit_duration"),
		t.String("commit_sha"),
		t.String("commit_branch"),
		t.String("commit_pr"),
		t.String("commit_author"),
		t.String("commit_gravatar"),
		t.String("commit_timestamp"),
		t.String("commit_message"),
		t.Varchar("commit_yaml", 4000),
		t.Integer("commit_created"),
		t.Integer("commit_updated"),
	}); err != nil {
		return err
	}

	_, err := mg.AddIndex("commits", []string{"commit_sha", "commit_branch", "repo_id"}, "unique")
	return err
}

func (m *Migrate_201408161536) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("commits")
	return err
}
