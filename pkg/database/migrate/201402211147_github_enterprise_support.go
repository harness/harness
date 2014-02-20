package migrate

type Rev3 struct{}

var GitHubEnterpriseSupport = &Rev3{}

func (r *Rev3) Revision() int64 {
	return 201402211147
}

func (r *Rev3) Up(op Operation) error {
	_, err := op.AddColumn("settings", "github_domain VARCHAR(255)")
	if err != nil {
		return err
	}
	_, err = op.AddColumn("settings", "github_apiurl VARCHAR(255)")

	op.Exec("update settings set github_domain=?", "github.com")
	op.Exec("update settings set github_apiurl=?", "https://api.github.com")
	return err
}

func (r *Rev3) Down(op Operation) error {
	_, err := op.DropColumns("settings", []string{"github_domain", "github_apiurl"})
	return err
}
