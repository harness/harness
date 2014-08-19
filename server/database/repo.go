package database

import (
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

type RepoManager interface {
	// Find retrieves the Repo by ID.
	Find(id int64) (*model.Repo, error)

	// FindName retrieves the Repo by the remote, owner and name.
	FindName(remote, owner, name string) (*model.Repo, error)

	// Insert persists a new Repo to the datastore.
	Insert(repo *model.Repo) error

	// Insert persists a modified Repo to the datastore.
	Update(repo *model.Repo) error

	// Delete removes a Repo from the datastore.
	Delete(repo *model.Repo) error

	// List retrieves all repositories from the datastore.
	List(user int64) ([]*model.Repo, error)

	// List retrieves all public repositories from the datastore.
	//ListPublic(user int64) ([]*Repo, error)
}

// A list of repositories when user has access
const repoListQuery = `
SELECT repos.* 
FROM repos 
JOIN perms p ON p.repo_id = repos.id
	WHERE p.user_id = ?
	AND p.read = '1'
ORDER BY repos.id ASC
`

func NewRepoManager(db *gorm.DB) RepoManager {
	return &repoManager{ORM: db}
}

type repoManager struct {
	ORM *gorm.DB
}

func (db *repoManager) Find(id int64) (*model.Repo, error) {
	repo := model.Repo{}

	err := db.ORM.Table("repos").Where(&model.Repo{Id: id}).First(&repo).Error
	return &repo, err
}

func (db *repoManager) FindName(host, owner, name string) (*model.Repo, error) {
	repo := model.Repo{}

	err := db.ORM.Table("repos").Where(&model.Repo{Host: host, Owner: owner, Name: name}).First(&repo).Error
	return &repo, err
}

func (db *repoManager) List(user int64) ([]*model.Repo, error) {
	var repos []*model.Repo

	// Get all permited repos
	err := db.ORM.Raw(repoListQuery, user).Find(&repos).Error

	return repos, err
}

func (db *repoManager) Insert(repo *model.Repo) error {
	repo.Created = time.Now().Unix()
	repo.Updated = time.Now().Unix()

	return db.ORM.Table("repos").Create(repo).Error
}

func (db *repoManager) Update(repo *model.Repo) error {
	repo.Updated = time.Now().Unix()

	// Fix bool update https://github.com/jinzhu/gorm/issues/202#issuecomment-52582525
	return db.ORM.Table("repos").Where(model.Repo{Id: repo.Id}).Update(repo).
		Update(map[string]interface{}{
		"Active":      repo.Active,
		"Private":     repo.Private,
		"PostCommit":  repo.PostCommit,
		"PullRequest": repo.PullRequest,
		"Privileged":  repo.Privileged}).Error
}

func (db *repoManager) Delete(repo *model.Repo) error {
	return db.ORM.Table("repos").Delete(repo).Error
}
