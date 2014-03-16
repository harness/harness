package migrate

type Rev3 struct{}

var GitHubEnterpriseSupport = &Rev3{}

func (r *Rev3) Revision() int64 {
	return 201402211147
}

func (r *Rev3) Up(mg *MigrationDriver) error {
	_, err := mg.AddColumn("settings", "github_domain VARCHAR(255)")
	if err != nil {
		return err
	}
	_, err = mg.AddColumn("settings", "github_apiurl VARCHAR(255)")

	mg.Tx.Exec("update settings set github_domain=?", "github.com")
	mg.Tx.Exec("update settings set github_apiurl=?", "https://api.github.com")
	return err
}

func (r *Rev3) Down(mg *MigrationDriver) error {
	_, err := mg.DropColumns("settings", "github_domain", "github_apiurl")
	return err
}
