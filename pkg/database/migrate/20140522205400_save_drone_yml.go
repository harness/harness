package migrate

type rev20140522205400 struct{}

var SaveDroneYml = &rev20140522205400{}

func (r *rev20140522205400) Revision() int64 {
	return 20140522205400
}

func (r *rev20140522205400) Up(mg *MigrationDriver) error {
	_, err := mg.AddColumn("builds", "buildscript TEXT")
	_, err = mg.Tx.Exec("UPDATE builds SET buildscript = '' WHERE buildscript IS NULL")
	return err
}

func (r *rev20140522205400) Down(mg *MigrationDriver) error {
	_, err := mg.DropColumns("builds", "buildscript")
	return err
}
