package migrate

type Rev1 struct{}

var RenamePrivelegedToPrivileged = &Rev1{}

func (r *Rev1) Revision() int64 {
	return 201402200603
}

func (r *Rev1) Up(mg *MigrationDriver) error {
	_, err := mg.RenameColumns("repos", map[string]string{
		"priveleged": "privileged",
	})
	return err
}

func (r *Rev1) Down(mg *MigrationDriver) error {
	_, err := mg.RenameColumns("repos", map[string]string{
		"privileged": "priveleged",
	})
	return err
}
