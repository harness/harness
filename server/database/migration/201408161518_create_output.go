package migration

import (
	"github.com/jinzhu/gorm"
)

type Output struct {
	Id        int64 `gorm:"primary_key:yes"`
	CommitId  int64
	OutputRaw string
}

type Migrate_201408161518 struct{}

var CreateOutputTable = &Migrate_201408161518{}

func (m *Migrate_201408161518) Revision() int64 {
	return 201408161518
}

func (m *Migrate_201408161518) Up(tx *gorm.DB) error {
	if err := tx.Table("output").CreateTable(Output{}).Error; err != nil {
		return err
	}

	err := tx.Table("output").AddUniqueIndex("idx_output_commit_id", "commit_id").Error
	return err
}

func (m *Migrate_201408161518) Down(tx *gorm.DB) error {
	return tx.Table("output").DropTable(Output{}).Error
}
