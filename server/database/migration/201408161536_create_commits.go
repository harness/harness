package migration

import (
	"github.com/jinzhu/gorm"
)

type Commit struct {
	Id          int64 `gorm:"primary_key:yes"`
	RepoId      int64
	Status      string
	Started     int64
	Finished    int64
	Duration    int64
	Sha         string
	Branch      string
	PullRequest string
	Author      string
	Gravatar    string
	Timestamp   string
	Message     string
	Config      string
	Created     int64
	Updated     int64
}

type Migrate_201408161536 struct{}

var CreateCommitsTable = &Migrate_201408161536{}

func (m *Migrate_201408161536) Revision() int64 {
	return 201408161536
}

func (m *Migrate_201408161536) Up(tx *gorm.DB) error {
	if err := tx.CreateTable(Commit{}).Error; err != nil {
		return err
	}

	err := tx.Model(Commit{}).AddUniqueIndex("idx_commit_sha_branch_repo_id", "sha", "branch", "repo_id").Error
	return err
}

func (m *Migrate_201408161536) Down(tx *gorm.DB) error {
	return tx.DropTable(Commit{}).Error
}
