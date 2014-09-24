package migrationutil

import (
	"database/sql"
	"log"

	"github.com/drone/drone/server/helper"
)

const migrationTableStmt = `
CREATE TABLE IF NOT EXISTS drone_migrations (
	revision BIGINT PRIMARY KEY
)
`

const migrationSelectStmt = `
SELECT revision FROM drone_migrations
WHERE revision = ?
`

const migrationSelectMaxStmt = `
SELECT max(revision) FROM drone_migrations
`

const insertRevisionStmt = `
INSERT INTO drone_migrations (revision) VALUES (?)
`

const deleteRevisionStmt = `
DELETE FROM drone_migrations where revision = ?
`

type Revision interface {
	Up(mg *MigrationDriver) error
	Down(mg *MigrationDriver) error
	Revision() int64
}

type Migration struct {
	db   *sql.DB
	revs []Revision
}

var Driver DriverBuilder

func New(db *sql.DB) *Migration {
	return &Migration{db: db}
}

// Add the Revision to the list of migrations.
func (m *Migration) Add(rev ...Revision) *Migration {
	m.revs = append(m.revs, rev...)
	return m
}

// Migrate executes the full list of migrations.
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

// MigrateTo executes all database migration until
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

	mg := Driver(tx)

	// loop through and execute revisions
	for _, rev := range m.revs {
		if rev.Revision() > current && rev.Revision() <= target {
			current = rev.Revision()
			// execute the revision Upgrade.
			if err := rev.Up(mg); err != nil {

				log.Printf("Failed to upgrade to Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback()
			}
			// update the revision number in the database
			if _, err := tx.Exec(helper.Rebind(insertRevisionStmt), current); err != nil {
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

	mg := Driver(tx)

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
			if err := rev.Down(mg); err != nil {
				log.Printf("Failed to downgrade from Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback()
			}
			// update the revision number in the database
			if _, err := tx.Exec(helper.Rebind(deleteRevisionStmt), current); err != nil {
				log.Printf("Failed to unregistser Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback()
			}

			log.Printf("Successfully downgraded from Revision %v\n", current)
		}
	}

	return tx.Commit()
}
