package migrationutil

import (
	"github.com/drone/drone/server/database/migration"
)

// All is called to collect all migration scripts
// and adds them to Revision list. New Revision
// should be added here ordered by its revision
// number.
func (m *Migration) All() *Migration {
	m.Add(migration.CreateUsersTable)
	m.Add(migration.CreateReposTable)
	m.Add(migration.CreateOutputTable)
	m.Add(migration.CreateServersTable)
	m.Add(migration.CreateSMTPTable)
	m.Add(migration.CreateRemotesTable)
	m.Add(migration.CreateCommitsTable)
	m.Add(migration.CreatePermsTable)
	return m
}
