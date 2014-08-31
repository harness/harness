package migrationutil

import (
	"log"

	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

const migrationTableStmt = `
CREATE TABLE IF NOT EXISTS migration (
	revision BIGINT PRIMARY KEY
)
`

const migrationSelectMaxStmt = `
SELECT MAX(revision) FROM migration
`

type Revision interface {
	Up(db *gorm.DB) error
	Down(db *gorm.DB) error
	Revision() int64
}

type Migration struct {
	DB         *gorm.DB
	Revisions  []Revision
	LogEnabled bool
}

func New(db *gorm.DB, log bool) *Migration {
	return &Migration{DB: db, LogEnabled: log}
}

func (m *Migration) Add(rev ...Revision) *Migration {
	m.Revisions = append(m.Revisions, rev...)
	return m
}

func (m *Migration) Migrate() error {
	var target int64
	if len(m.Revisions) > 0 {
		target = m.Revisions[len(m.Revisions)-1].Revision()
	}
	return m.MigrateTo(target)
}

func (m *Migration) MigrateTo(target int64) error {
	if err := m.DB.Exec(migrationTableStmt).Error; err != nil {
		return err
	}

	var current int64
	row := m.DB.Table("migration").Select("MAX(revision)").Row()
	row.Scan(&current)

	if target < current {
		return m.down(target, current)
	}

	return m.up(target, current)
}

func (m *Migration) up(target, current int64) error {
	tx := m.DB.Begin()

	for _, rev := range m.Revisions {
		if rev.Revision() > current && rev.Revision() <= target {
			current = rev.Revision()

			if err := rev.Up(tx); err != nil {
				log.Printf("Failed to upgrade to Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback().Error
			}

			if err := tx.Table("migration").Create(model.Migrate{Revision: current}).Error; err != nil {
				log.Printf("Failed to register Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback().Error
			}

			if m.LogEnabled {
				log.Printf("Successfully upgraded to Revision %v\n", current)
			}
		}
	}

	return tx.Commit().Error
}

func (m *Migration) down(target, current int64) error {
	// create the database transaction
	tx := m.DB.Begin()

	// reverse the list of revisions
	revs := []Revision{}
	for _, rev := range m.Revisions {
		revs = append([]Revision{rev}, revs...)
	}

	// loop through the (reversed) list of
	// revisions and execute.
	for _, rev := range revs {
		if rev.Revision() > target {
			current = rev.Revision()
			// execute the revision Upgrade.
			if err := rev.Down(tx); err != nil {
				log.Printf("Failed to downgrade from Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback().Error
			}
			// update the revision number in the database
			if err := tx.Table("migration").Delete(model.Migrate{Revision: current}).Error; err != nil {
				log.Printf("Failed to unregistser Revision Number %v\n", current)
				log.Println(err)
				return tx.Rollback().Error
			}

			log.Printf("Successfully downgraded from Revision %v\n", current)
		}
	}

	return tx.Commit().Error
}
