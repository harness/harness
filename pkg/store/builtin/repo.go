package builtin

import (
	"database/sql"
	"time"

	"github.com/drone/drone/pkg/types"
)

type Repostore struct {
	*sql.DB
}

func NewRepostore(db *sql.DB) *Repostore {
	return &Repostore{db}
}

// Repo retrieves a specific repo from the
// datastore for the given ID.
func (db *Repostore) Repo(id int64) (*types.Repo, error) {
	return getRepo(db, rebind(stmtRepoSelect), id)
}

// RepoName retrieves a repo from the datastore
// for the specified name.
func (db *Repostore) RepoName(owner, name string) (*types.Repo, error) {
	return getRepo(db, rebind(stmtRepoSelectRepoOwnerName), owner, name)
}

// RepoList retrieves a list of all repos from
// the datastore accessible by the given user ID.
func (db *Repostore) RepoList(user *types.User) ([]*types.Repo, error) {
	return getRepos(db, rebind(repoListQuery), user.ID)
}

// AddRepo inserts a repo in the datastore.
func (db *Repostore) AddRepo(repo *types.Repo) error {
	repo.Created = time.Now().UTC().Unix()
	repo.Updated = time.Now().UTC().Unix()
	return createRepo(db, rebind(stmtRepoInsert), repo)
}

// SetRepo updates a repo in the datastore.
func (db *Repostore) SetRepo(repo *types.Repo) error {
	repo.Updated = time.Now().UTC().Unix()
	return updateRepo(db, rebind(stmtRepoUpdate), repo)
}

// DelRepo removes the repo from the datastore.
func (db *Repostore) DelRepo(repo *types.Repo) error {
	var _, err = db.Exec(rebind(stmtRepoDelete), repo.ID)
	return err
}

// SQL statement to retrieve a list of Repos
// with permissions for the given User ID.
const repoListQuery = `
SELECT
 r.repo_id
,r.repo_user_id
,r.repo_owner
,r.repo_name
,r.repo_full_name
,r.repo_token
,r.repo_language
,r.repo_private
,r.repo_self
,r.repo_link
,r.repo_clone
,r.repo_branch
,r.repo_timeout
,r.repo_trusted
,r.repo_post_commit
,r.repo_pull_request
,r.repo_public_key
,r.repo_private_key
,r.repo_created
,r.repo_updated
,r.repo_params
FROM
 repos r
,stars s
WHERE r.repo_id = s.repo_id
  AND s.user_id = ?
`
