package migrationutil

type Migrate_201408161213 struct{}

var CreateUsersTable = &Migrate_201408161213{}

func (m *Migrate_201408161213) Revision() int64 {
	return 201408161213
}

func (m *Migrate_201408161213) Up(mg *MigrationDriver) error {
	t := mg.T

	if _, err := mg.CreateTable("users", []string{
		t.Pk("user_id"),
		t.String("user_remote"),
		t.String("user_login"),
		t.String("user_access"),
		t.String("user_secret"),
		t.String("user_name"),
		t.String("user_email"),
		t.String("user_gravatar"),
		t.String("user_token"),
		t.Bool("user_admin"),
		t.Bool("user_active"),
		t.Bool("user_syncing"),
		t.Integer("user_created"),
		t.Integer("user_updated"),
		t.Integer("user_synced"),
	}); err != nil {
		return err
	}

	if _, err := mg.AddIndex("users", []string{"user_remote", "user_login"}, "unique"); err != nil {
		return err
	}

	_, err := mg.AddIndex("users", []string{"user_token"}, "unique")
	return err

}

func (m *Migrate_201408161213) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("users")
	return err
}
