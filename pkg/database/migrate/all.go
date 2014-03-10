package migrate

func (m *Migration) All() *Migration {

	// List all migrations here
	m.Add(SetupTables)
	m.Add(SetupIndices)
	m.Add(RenamePrivelegedToPrivileged)
	m.Add(GitHubEnterpriseSupport)
	m.Add(AddOpenInvitationColumn)

	// m.Add(...)
	// ...
	return m
}
