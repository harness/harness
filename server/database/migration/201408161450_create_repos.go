package migration

import (
	"github.com/jinzhu/gorm"
)

type Repo struct {
	Id     int64 `gorm:"primary_key:yes"        json:"-"`
	UserId int64
	Remote string
	Host   string
	Owner  string
	Name   string

	Url      string
	CloneUrl string
	GitUrl   string
	SshUrl   string

	Active      bool
	Private     bool
	Privileged  bool
	PostCommit  bool
	PullRequest bool
	PublicKey   string
	PrivateKey  string
	Params      string
	Timeout     int64
	Created     int64
	Updated     int64
}

type Migrate_201408161450 struct{}

var CreateReposTable = &Migrate_201408161450{}

func (m *Migrate_201408161450) Revision() int64 {
	return 201408161450
}

func (m *Migrate_201408161450) Up(tx *gorm.DB) error {
	if err := tx.CreateTable(Repo{}).Error; err != nil {
		return err
	}

	err := tx.Model(Repo{}).AddUniqueIndex("idx_repo_host_owner_name", "host", "owner", "name").Error
	return err
}

func (m *Migrate_201408161450) Down(tx *gorm.DB) error {
	return tx.DropTable(Repo{}).Error
}
