package migration

import (
	"github.com/jinzhu/gorm"
)

type Perm struct {
	Id      int64 `gorm:"primary_key:yes"`
	UserId  int64
	RepoId  int64
	Read    bool
	Write   bool
	Admin   bool
	Guest   bool `sql:"-"`
	Created int64
	Updated int64
}

type Migrate_201408161538 struct{}

var CreatePermsTable = &Migrate_201408161538{}

func (m *Migrate_201408161538) Revision() int64 {
	return 201408161538
}

func (m *Migrate_201408161538) Up(tx *gorm.DB) error {
	if err := tx.CreateTable(Perm{}).Error; err != nil {
		return nil
	}

	err := tx.Model(Perm{}).AddUniqueIndex("idx_perm_repo_id_user_id", "repo_id", "user_id").Error
	return err
}

func (m *Migrate_201408161538) Down(tx *gorm.DB) error {
	return tx.DropTable(Perm{}).Error
}
