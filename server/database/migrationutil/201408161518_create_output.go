package migrationutil

type Migrate_201408161518 struct{}

var CreateOutputTable = &Migrate_201408161518{}

func (m *Migrate_201408161518) Revision() int64 {
	return 201408161518
}

func (m *Migrate_201408161518) Up(mg *MigrationDriver) error {
	t := mg.T

	if _, err := mg.CreateTable("output", []string{
		t.Pk("output_id"),
		t.Integer("commit_id"),
		t.Blob("output_raw"),
	}); err != nil {
		return err
	}

	_, err := mg.AddIndex("output", []string{"commit_id"}, "unique")
	return err
}

func (m *Migrate_201408161518) Down(mg *MigrationDriver) error {
	_, err := mg.DropTable("output")
	return err
}
