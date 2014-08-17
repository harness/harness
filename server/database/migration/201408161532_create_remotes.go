package migration

import (
	"github.com/jinzhu/gorm"
)

type Remote struct {
	Id     int64 `gorm:"primary_key:yes"`
	Type   string
	Host   string
	Url    string
	Api    string
	Client string
	Secret string
	Open   bool
}

type Migrate_201408161532 struct{}

var CreateRemotesTable = &Migrate_201408161532{}

func (m *Migrate_201408161532) Revision() int64 {
	return 201408161532
}

func (m *Migrate_201408161532) Up(tx *gorm.DB) error {
	if err := tx.CreateTable(Remote{}).Error; err != nil {
		return err
	}

	if err := tx.Model(Remote{}).AddUniqueIndex("idx_remote_host", "host").Error; err != nil {
		return err
	}

	err := tx.Model(Remote{}).AddUniqueIndex("idx_remote_type", "type").Error
	return err
}

func (m *Migrate_201408161532) Down(tx *gorm.DB) error {
	return tx.DropTable(Remote{}).Error
}
