// Usage
//    migrate.To(2)
//    	.Add(Version_1)
//    	.Add(Version_2)
//    	.Add(Version_3)
//    	.Exec(db)
//
//    migrate.ToLatest()
//    	.Add(Version_1)
//    	.Add(Version_2)
//    	.Add(Version_3)
//    	.SetDialect(migrate.MySQL)
//    	.Exec(db)
//
//    migrate.ToLatest()
//    	.Add(Version_1)
//    	.Add(Version_2)
//    	.Add(Version_3)
//		.Backup(path)
//		.Exec()

package migrate

import (
	"database/sql"
	"log"
)

const migrationTableStmt = `
CREATE TABLE IF NOT EXISTS migration (
	revision NUMBER PRIMARY KEY
)
`

const migrationSelectStmt = `
SELECT revision FROM migration
WHERE revision = ?
`

const migrationSelectMaxStmt = `
SELECT max(revision) FROM migration
`

const insertRevisionStmt = `
INSERT INTO migration (revision) VALUES (?)
`

const deleteRevisionStmt = `
DELETE FROM migration where revision = ?
`

// Operation interface covers basic migration operations.
// Implementation details is specific for each database,
// see migrate/sqlite.go for implementation reference.
type Operation interface {

	CreateTable(tableName string, args []string) (sql.Result, error)

	RenameTable(tableName, newName string) (sql.Result, error)

	DropTable(tableName string) (sql.Result, error)

	AddColumn(tableName, columnSpec string) (sql.Result, error)

	DropColumns(tableName string, columnsToDrop []string) (sql.Result, error)

	RenameColumns(tableName string, columnChanges map[string]string) (sql.Result, error)
}

type Revision interface {
	Up(op Operation) error
	Down(op Operation) error
	Revision() int64
}

type MigrationDriver struct {
	Tx *sql.Tx
}

type Migration struct {
	db   *sql.DB
	revs []Revision
}

var Driver func(tx *sql.Tx) Operation

func New(db *sql.DB) *Migration {
	return &Migration{db: db}
}

// Add the Revision to the list of migrations.
func (m *Migration) Add(rev ...Revision) *Migration {
	m.revs = append(m.revs, rev...)
	return m
}

// Execute the full list of migrations.
func (m *Migration) Migrate() error {
	var target int64
	if len(m.revs) > 0 {
		// get the last revision number in
		// the list. This is what we'll
		// migrate toward.
		target = m.revs[len(m.revs)-1].Revision()
	}
	return m.MigrateTo(target)
}

// Execute all database migration until
// you are at the specified revision number.
// If the revision number is less than the
// current revision, then we will downgrade.
func (m *Migration) MigrateTo(target int64) error {

	// make sure the migration table is created.
	if _, err := m.db.Exec(migrationTableStmt); err != nil {
		return err
	}

	// get the current revision
	var current int64
	m.db.QueryRow(migrationSelectMaxStmt).Scan(&current)

	// already up to date
	if current == target {
		log.Println("Database already up-to-date.")
		return nil
	}

	// should we downgrade?
	if target < current {
		return m.down(target, current)
	}

	// else upgrade
	return m.up(target, current)
}

func (m *Migration) up(target, current int64) error {
	// create the database transaction
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	op := Driver(tx)

	// loop through and execute revisions
	for _, rev := range m.revs {
		if rev.Revision() >= target {
			current = rev.Revision()
			// execute the revision Upgrade.
			if err := rev.Up(op); err != nil {
				log.Printf("Failed to upgrade to Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback()
			}
			// update the revision number in the database
			if _, err := tx.Exec(insertRevisionStmt, current); err != nil {
				log.Printf("Failed to register Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback()
			}

			log.Printf("Successfully upgraded to Revision %v\n", current)
		}
	}

	return tx.Commit()
}

func (m *Migration) down(target, current int64) error {
	// create the database transaction
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	op := Driver(tx)

	// reverse the list of revisions
	revs := []Revision{}
	for _, rev := range m.revs {
		revs = append([]Revision{rev}, revs...)
	}

	// loop through the (reversed) list of
	// revisions and execute.
	for _, rev := range revs {
		if rev.Revision() > target {
			current = rev.Revision()
			// execute the revision Upgrade.
			if err := rev.Down(op); err != nil {
				log.Printf("Failed to downgrade to Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback()
			}
			// update the revision number in the database
			if _, err := tx.Exec(deleteRevisionStmt, current); err != nil {
				log.Printf("Failed to unregistser Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback()
			}

			log.Printf("Successfully downgraded to Revision %v\n", current)
		}
	}

	return tx.Commit()
}
