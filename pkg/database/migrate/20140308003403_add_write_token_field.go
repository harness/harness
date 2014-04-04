package migrate

type rev20140308003403 struct{}

var AddWriteTokenField = &rev20140308003403{}

func (r *rev20140308003403) Revision() int64 {
return 20140308003403
}

func (r *rev20140308003403) Up(mg *MigrationDriver) error {
	_, err := mg.AddColumn("users", "github_write_token VARCHAR(255)")
	return err
}

func (r *rev20140308003403) Down(mg *MigrationDriver) error {
	_, err := mg.DropColumns("users", "github_write_token")
	return err
}
