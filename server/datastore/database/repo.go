package database

import (
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type Repostore struct {
	meddler.DB
}

func NewRepostore(db meddler.DB) *Repostore {
	return &Repostore{db}
}

// GetRepo retrieves a specific repo from the
// datastore for the given ID.
func (db *Repostore) GetRepo(id int64) (*model.Repo, error) {
	var repo = new(model.Repo)
	var err = meddler.Load(db, repoTable, repo, id)
	return repo, err
}

// GetRepoName retrieves a repo from the datastore
// for the specified remote, owner and name.
func (db *Repostore) GetRepoName(remote, owner, name string) (*model.Repo, error) {
	var repo = new(model.Repo)
	var err = meddler.QueryRow(db, repo, rebind(repoNameQuery), remote, owner, name)
	return repo, err
}

// GetRepoList retrieves a list of all repos from
// the datastore accessible by the given user ID.
func (db *Repostore) GetRepoList(user *model.User) ([]*model.Repo, error) {
	var repos []*model.Repo
	var err = meddler.QueryAll(db, &repos, rebind(repoListQuery), user.ID)
	return repos, err
}

// PostRepo saves a repo in the datastore.
func (db *Repostore) PostRepo(repo *model.Repo) error {
	if repo.Created == 0 {
		repo.Created = time.Now().UTC().Unix()
	}
	repo.Updated = time.Now().UTC().Unix()
	return meddler.Save(db, repoTable, repo)
}

// PutRepo saves a repo in the datastore.
func (db *Repostore) PutRepo(repo *model.Repo) error {
	if repo.Created == 0 {
		repo.Created = time.Now().UTC().Unix()
	}
	repo.Updated = time.Now().UTC().Unix()
	return meddler.Save(db, repoTable, repo)
}

// DelRepo removes the repo from the datastore.
func (db *Repostore) DelRepo(repo *model.Repo) error {
	var _, err = db.Exec(rebind(repoDeleteStmt), repo.ID)
	return err
}

// Repo table name in database.
const repoTable = "repos"

// SQL statement to retrieve a Repo by name.
const repoNameQuery = `
SELECT *
FROM repos
WHERE repo_host  = ?
  AND repo_owner = ?
  AND repo_name  = ?
LIMIT 1;
`

// SQL statement to retrieve a list of Repos
// with permissions for the given User ID.
const repoListQuery = `
SELECT r.*
FROM
 repos r
,perms p
WHERE r.repo_id = p.repo_id
  AND p.user_id = ?
`

// SQL statement to delete a User by ID.
const repoDeleteStmt = `
DELETE FROM repos
WHERE repo_id = ?
`
