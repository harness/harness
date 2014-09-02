package database

import (
	"database/sql"
	"time"

	"github.com/drone/drone/server/helper"
	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
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

func NewRepoManager(db *sql.DB) RepoManager {
	return &repoManager{db}
}

type repoManager struct {
	*sql.DB
}

func (db *repoManager) Find(id int64) (*model.Repo, error) {
	const query = "select * from repos where repo_id = ?"
	var repo = model.Repo{}
	var err = meddler.QueryRow(db, &repo, helper.Rebind(query), id)
	return &repo, err
}

func (db *repoManager) FindName(remote, owner, name string) (*model.Repo, error) {
	const query = "select * from repos where repo_host = ? and repo_owner = ? and repo_name = ?"
	var repo = model.Repo{}
	var err = meddler.QueryRow(db, &repo, helper.Rebind(query), remote, owner, name)
	return &repo, err
}

func (db *repoManager) List(user int64) ([]*model.Repo, error) {
	const query = "select * from repos where repo_id IN (select repo_id from perms where user_id = ?)"
	var repos []*model.Repo
	err := meddler.QueryAll(db, &repos, helper.Rebind(query), user)
	return repos, err
}

func (db *repoManager) Insert(repo *model.Repo) error {
	repo.Created = time.Now().Unix()
	repo.Updated = time.Now().Unix()
	return meddler.Insert(db, "repos", repo)
}

func (db *repoManager) Update(repo *model.Repo) error {
	repo.Updated = time.Now().Unix()
	return meddler.Update(db, "repos", repo)
}

func (db *repoManager) Delete(repo *model.Repo) error {
	const stmt = "delete from repos where repo_id = ?"
	_, err := db.Exec(helper.Rebind(stmt), repo.ID)
	return err
}
