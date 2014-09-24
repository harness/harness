package migrationutil

type Migrate_201408161532 struct{}

var CreateRemotesTable = &Migrate_201408161532{}

func (m *Migrate_201408161532) Revision() int64 {
	return 201408161532
}

func (m *Migrate_201408161532) Up(mg *MigrationDriver) error {
	t := mg.T

	if _, err := mg.CreateTable("remotes", []string{
		t.Pk("remote_id"),
		t.String("remote_type"),
		t.String("remote_host"),
		t.String("remote_url"),
		t.String("remote_api"),
		t.String("remote_client"),
		t.String("remote_secret"),
		t.Bool("remote_open"),
	}); err != nil {
		return err
	}

	if _, err := mg.AddIndex("remotes", []string{"remote_type"}, "unique"); err != nil {
		return err
	}

	_, err := mg.AddIndex("remotes", []string{"remote_host"}, "unique")
	return err
}

func (m *Migrate_201408161532) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("remotes")
	return err
}
