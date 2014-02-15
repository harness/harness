package migrate

func (m *Migration) All() *Migration {

	// List all migrations here
	m.Add(RenamePrivelegedToPrivileged)

	// m.Add(...)
	// ...
	return m
}
