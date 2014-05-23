package migrate

// All is called to collect all migration scripts
// and adds them to Revision list. New Revision
// should be added here ordered by its revision
// number.
func (m *Migration) All() *Migration {

	// List all migrations here
	m.Add(SetupTables)
	m.Add(SetupIndices)
	m.Add(RenamePrivelegedToPrivileged)
	m.Add(GitHubEnterpriseSupport)
	m.Add(AddOpenInvitationColumn)
	m.Add(AddGitlabColumns)
	m.Add(SaveDroneYml)

	// m.Add(...)
	// ...
	return m
}
