package repo

import (
	"database/sql"
	"time"

	"github.com/russross/meddler"
)

type RepoManager interface {
	// Find retrieves the Repo by ID.
	Find(id int64) (*Repo, error)

	// FindName retrieves the Repo by the remote, owner and name.
	FindName(remote, owner, name string) (*Repo, error)

	// Insert persists a new Repo to the datastore.
	Insert(repo *Repo) error

	// Insert persists a modified Repo to the datastore.
	Update(repo *Repo) error

	// Delete removes a Repo from the datastore.
	Delete(repo *Repo) error

	// List retrieves all repositories from the datastore.
	List(user int64) ([]*Repo, error)

	// List retrieves all public repositories from the datastore.
	//ListPublic(user int64) ([]*Repo, error)
}

func NewManager(db *sql.DB) RepoManager {
	return &repoManager{db}
}

type repoManager struct {
	*sql.DB
}

func (db *repoManager) Find(id int64) (*Repo, error) {
	const query = "select * from repos where repo_id = ?"
	var repo = Repo{}
	var err = meddler.QueryRow(db, &repo, query, id)
	return &repo, err
}

func (db *repoManager) FindName(remote, owner, name string) (*Repo, error) {
	const query = "select * from repos where repo_host = ? and repo_owner = ? and repo_name = ?"
	var repo = Repo{}
	var err = meddler.QueryRow(db, &repo, query, remote, owner, name)
	return &repo, err
}

func (db *repoManager) List(user int64) ([]*Repo, error) {
	const query = "select * from repos where repo_id IN (select repo_id from perms where user_id = ?)"
	var repos []*Repo
	err := meddler.QueryAll(db, &repos, query, user)
	return repos, err
}

//func (db *repoManager) ListPublic(user int64) ([]*Repo, error) {
//	const query = "select * from repos where repo_id IN (select repo_id from perms where user_id = ?) AND repo_private=0"
//	var repos []*Repo
//	err := meddler.QueryAll(db, &repos, query, user)
//	return repos, err
//}

func (db *repoManager) Insert(repo *Repo) error {
	repo.Created = time.Now().Unix()
	repo.Updated = time.Now().Unix()
	return meddler.Insert(db, "repos", repo)
}

func (db *repoManager) Update(repo *Repo) error {
	repo.Updated = time.Now().Unix()
	return meddler.Update(db, "repos", repo)
}

func (db *repoManager) Delete(repo *Repo) error {
	const stmt = "delete from repos where repo_id = ?"
	_, err := db.Exec(stmt, repo.ID)
	return err
}
