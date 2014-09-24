package migrationutil

// All is called to collect all migration scripts
// and adds them to Revision list. New Revision
// should be added here ordered by its revision
// number.
func (m *Migration) All() *Migration {
	m.Add(CreateUsersTable)
	m.Add(CreateReposTable)
	m.Add(CreateOutputTable)
	m.Add(CreateServersTable)
	m.Add(CreateSMTPTable)
	m.Add(CreateRemotesTable)
	m.Add(CreateCommitsTable)
	m.Add(CreatePermsTable)
	return m
}
