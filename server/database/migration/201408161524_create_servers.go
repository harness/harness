package migration

import (
	"github.com/jinzhu/gorm"
)

type Server struct {
	Id   int64  `gorm:"primary_key:yes" json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`
	User string `json:"user"`
	Pass string `json:"name"`
	Cert string `json:"cert"`
}

type Migrate_201408161524 struct{}

var CreateServersTable = &Migrate_201408161524{}

func (m *Migrate_201408161524) Revision() int64 {
	return 201408161524
}

func (m *Migrate_201408161524) Up(tx *gorm.DB) error {
	if err := tx.CreateTable(Server{}).Error; err != nil {
		return err
	}

	err := tx.Model(Server{}).AddUniqueIndex("idx_server_name", "name").Error
	return err
}

func (m *Migrate_201408161524) Down(tx *gorm.DB) error {
	return tx.DropTable(Server{}).Error
}
