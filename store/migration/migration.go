package migration

//go:generate go-bindata -pkg migration -o migration_gen.go sqlite3/ mysql/ postgres/
