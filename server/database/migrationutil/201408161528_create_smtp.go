package migrationutil

type Migrate_201408161528 struct{}

var CreateSMTPTable = &Migrate_201408161528{}

func (m *Migrate_201408161528) Revision() int64 {
	return 201408161528
}

func (m *Migrate_201408161528) Up(mg *MigrationDriver) error {
	t := mg.T

	_, err := mg.CreateTable("smtp", []string{
		t.Pk("smtp_id"),
		t.String("smtp_from"),
		t.String("smtp_host"),
		t.String("smtp_port"),
		t.String("smtp_user"),
		t.String("smtp_pass"),
	})
	return err
}

func (m *Migrate_201408161528) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("smtp")
	return err
}
