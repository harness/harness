package migration

import (
	"github.com/jinzhu/gorm"
)

type SMTPServer struct {
	Id   int64  `gorm:"primary_key:yes" json:"id"`
	From string `json:"from"`
	Host string `json:"host"`
	Port string `json:"port"`
	User string `json:"user"`
	Pass string `json:"name"`
}

type Migrate_201408161528 struct{}

var CreateSMTPTable = &Migrate_201408161528{}

func (m *Migrate_201408161528) Revision() int64 {
	return 201408161528
}

func (m *Migrate_201408161528) Up(tx *gorm.DB) error {
	return tx.Table("smtp").CreateTable(SMTPServer{}).Error
}

func (m *Migrate_201408161528) Down(tx *gorm.DB) error {
	return tx.Table("smtp").DropTable(SMTPServer{}).Error
}
