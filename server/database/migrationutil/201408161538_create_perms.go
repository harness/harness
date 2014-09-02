package migrationutil

type Migrate_201408161538 struct{}

var CreatePermsTable = &Migrate_201408161538{}

func (m *Migrate_201408161538) Revision() int64 {
	return 201408161538
}

func (m *Migrate_201408161538) Up(mg *MigrationDriver) error {
	t := mg.T

	if _, err := mg.CreateTable("perms", []string{
		t.Pk("perm_id"),
		t.Integer("user_id"),
		t.Integer("repo_id"),
		t.Bool("perm_read"),
		t.Bool("perm_write"),
		t.Bool("perm_admin"),
		t.Integer("perm_created"),
		t.Integer("perm_updated"),
	}); err != nil {
		return err
	}

	_, err := mg.AddIndex("perms", []string{"repo_id", "user_id"}, "unique")
	return err
}

func (m *Migrate_201408161538) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("perms")
	return err
}
