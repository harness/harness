package builtin

import (
	"database/sql"
	"time"

	common "github.com/drone/drone/pkg/types"
	"github.com/russross/meddler"
)

type Repostore struct {
	*sql.DB
}

func NewRepostore(db *sql.DB) *Repostore {
	return &Repostore{db}
}

// Repo retrieves a specific repo from the
// datastore for the given ID.
func (db *Repostore) Repo(id int64) (*common.Repo, error) {
	var repo = new(common.Repo)
	var err = meddler.Load(db, repoTable, repo, id)
	return repo, err
}

// RepoName retrieves a repo from the datastore
// for the specified name.
func (db *Repostore) RepoName(owner, name string) (*common.Repo, error) {
	var repo = new(common.Repo)
	var err = meddler.QueryRow(db, repo, rebind(repoNameQuery), owner, name)
	return repo, err
}

// RepoList retrieves a list of all repos from
// the datastore accessible by the given user ID.
func (db *Repostore) RepoList(user *common.User) ([]*common.Repo, error) {
	var repos []*common.Repo
	var err = meddler.QueryAll(db, &repos, rebind(repoListQuery), user.ID)
	return repos, err
}

// AddRepo inserts a repo in the datastore.
func (db *Repostore) AddRepo(repo *common.Repo) error {
	repo.Created = time.Now().UTC().Unix()
	repo.Updated = time.Now().UTC().Unix()
	return meddler.Insert(db, repoTable, repo)
}

// SetRepo updates a repo in the datastore.
func (db *Repostore) SetRepo(repo *common.Repo) error {
	repo.Updated = time.Now().UTC().Unix()
	return meddler.Update(db, repoTable, repo)
}

// DelRepo removes the repo from the datastore.
func (db *Repostore) DelRepo(repo *common.Repo) error {
	var _, err = db.Exec(rebind(repoDeleteStmt), repo.ID)
	return err
}

// Repo table names in database.
const (
	repoTable      = "repos"
	repoKeyTable   = "repo_keys"
	repoParamTable = "repo_params"
)

// SQL statement to retrieve a Repo by name.
const repoNameQuery = `
SELECT *
FROM repos
WHERE repo_owner = ?
  AND repo_name  = ?
LIMIT 1;
`

// SQL statement to retrieve a list of Repos
// with permissions for the given User ID.
const repoListQuery = `
SELECT r.*
FROM
 repos r
,stars s
WHERE r.repo_id = s.repo_id
  AND s.user_id = ?
`

// SQL statement to retrieve a keypair for
// a Repository.
const repoKeysQuery = `
SELECT *
FROM repo_keys
WHERE repo_id = ?
LIMIT 1;
`

// SQL statement to retrieve a keypair for
// a Repository.
const repoParamsQuery = `
SELECT *
FROM repo_params
WHERE repo_id = ?
LIMIT 1;
`

// SQL statement to delete a User by ID.
const (
	repoDeleteStmt        = `DELETE FROM repos       WHERE repo_id = ?`
	repoKeypairDeleteStmt = `DELETE FROM repo_params WHERE repo_id = ?`
	repoParamsDeleteStmt  = `DELETE FROM repo_keys   WHERE repo_id = ?`
)
