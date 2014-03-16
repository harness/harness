package migrate

type rev20140310104446 struct{}

var AddOpenInvitationColumn = &rev20140310104446{}

func (r *rev20140310104446) Revision() int64 {
	return 20140310104446
}

func (r *rev20140310104446) Up(mg *MigrationDriver) error {
	// Suppress error here for backward compatibility
	_, err := mg.AddColumn("settings", "open_invitations BOOLEAN")
	_, err = mg.Tx.Exec("UPDATE settings SET open_invitations=0 WHERE open_invitations IS NULL")
	return err
}

func (r *rev20140310104446) Down(mg *MigrationDriver) error {
	_, err := mg.DropColumns("settings", "open_invitations")
	return err
}
