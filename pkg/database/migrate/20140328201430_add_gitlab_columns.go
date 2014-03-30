package migrate

type rev20140328201430 struct{}

var AddGitlabColumns = &rev20140328201430{}

func (r *rev20140328201430) Revision() int64 {
	return 20140328201430
}

func (r *rev20140328201430) Up(mg *MigrationDriver) error {
	// Migration steps here
	if _, err := mg.AddColumn("settings", mg.T.String("gitlab_domain")); err != nil {
		return err
	}
	if _, err := mg.AddColumn("settings", mg.T.String("gitlab_apiurl")); err != nil {
		return err
	}

	if _, err := mg.Tx.Exec(`update settings set gitlab_domain=?`, "gitlab.com"); err != nil {
		return err
	}

	if _, err := mg.Tx.Exec(`update settings set gitlab_apiurl=?`, "https://gitlab.com"); err != nil {
		return err
	}

	if _, err := mg.AddColumn("users", mg.T.String("gitlab_token")); err != nil {
		return err
	}

	_, err := mg.Tx.Exec(`update users set gitlab_token=?`, "")
	return err
}

func (r *rev20140328201430) Down(mg *MigrationDriver) error {
	// Revert migration steps here
	if _, err := mg.DropColumns("users", "gitlab_token"); err != nil {
		return err
	}
	_, err := mg.DropColumns("settings", "gitlab_domain", "gitlab_apiurl")
	return err
}
