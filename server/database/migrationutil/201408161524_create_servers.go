package migrationutil

type Migrate_201408161524 struct{}

var CreateServersTable = &Migrate_201408161524{}

func (m *Migrate_201408161524) Revision() int64 {
	return 201408161524
}

func (m *Migrate_201408161524) Up(mg *MigrationDriver) error {
	t := mg.T

	if _, err := mg.CreateTable("servers", []string{
		t.Pk("server_id"),
		t.String("server_name"),
		t.String("server_host"),
		t.String("server_user"),
		t.String("server_pass"),
		t.Varchar("server_cert", 4000),
	}); err != nil {
		return err
	}

	_, err := mg.AddIndex("servers", []string{"server_name"}, "unique")
	return err
}

func (m *Migrate_201408161524) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("servers")
	return err
}
