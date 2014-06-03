package migrate

type rev20140512013850 struct{}

var AddStashColumns = &rev20140512013850{}

func (r *rev20140512013850) Revision() int64 {
	return 20140512013850
}

func (r *rev20140512013850) Up(mg *MigrationDriver) error {
	// Migration steps here
	if _, err := mg.AddColumn("settings", mg.T.String("stash_domain")); err != nil {
		return err
	}
	if _, err := mg.AddColumn("settings", mg.T.String("stash_sshport")); err != nil {
		return err
	}
	if _, err := mg.AddColumn("settings", mg.T.String("stash_key")); err != nil {
		return err
	}
	if _, err := mg.AddColumn("settings", mg.T.String("stash_secret")); err != nil {
		return err
	}
	if _, err := mg.AddColumn("settings", mg.T.String("stash_hookkey")); err != nil {
		return err
	}

	if _, err := mg.AddColumn("users", mg.T.String("stash_login")); err != nil {
		return err
	}
	if _, err := mg.AddColumn("users", mg.T.String("stash_token")); err != nil {
		return err
	}
	if _, err := mg.AddColumn("users", mg.T.String("stash_secret")); err != nil {
		return err
	}
	if _, err := mg.AddColumn("settings", mg.T.String("stash_privatekey")); err != nil {
		return err
	}

	_, err := mg.Tx.Exec(`update users set stash_login=?`, "")
	_, err = mg.Tx.Exec(`update users set stash_token=?`, "")
	_, err = mg.Tx.Exec(`update users set stash_secret=?`, "")

	return err
}

func (r *rev20140512013850) Down(mg *MigrationDriver) error {
	// Revert migration steps here
	if _, err := mg.DropColumns("users", "stash_login", "stash_token", "stash_secret"); err != nil {
		return err
	}
	_, err := mg.DropColumns("settings", "stash_domain", "stash_key", "stash_sshport",
		"stash_secret", "stash_hookkey", "stash_privatekey")
	return err
}
