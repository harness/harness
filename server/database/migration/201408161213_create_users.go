package migration

import (
	"github.com/jinzhu/gorm"
)

type User struct {
	Id       int64 `gorm:"primary_key:yes"`
	Remote   string
	Login    string
	Access   string
	Secret   string
	Name     string
	Email    string
	Gravatar string
	Token    string
	Admin    bool
	Active   bool
	Syncing  bool
	Created  int64
	Updated  int64
	Synced   int64
}

type Migrate_201408161213 struct{}

var CreateUsersTable = &Migrate_201408161213{}

func (m *Migrate_201408161213) Revision() int64 {
	return 201408161213
}

func (m *Migrate_201408161213) Up(tx *gorm.DB) error {
	if err := tx.CreateTable(User{}).Error; err != nil {
		return err
	}

	if err := tx.Model(User{}).AddUniqueIndex("idx_user_token", "token").Error; err != nil {
		return err
	}

	err := tx.Model(User{}).AddUniqueIndex("idx_user_remote_login", "remote", "login").Error
	return err
}

func (m *Migrate_201408161213) Down(tx *gorm.DB) error {
	return tx.DropTable(User{}).Error
}
