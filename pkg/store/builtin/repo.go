package builtin

import (
	"database/sql"

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
	return createRepo(db, rebind(stmtRepoInsert), repo)
}

// SetRepo updates a repo in the datastore.
func (db *Repostore) SetRepo(repo *types.Repo) error {
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
 repo_id
,repo_user_id
,repo_owner
,repo_name
,repo_full_name
,repo_avatar
,repo_self
,repo_link
,repo_clone
,repo_branch
,repo_private
,repo_trusted
,repo_timeout
,repo_keys_public
,repo_keys_private
,repo_hooks_pull_request
,repo_hooks_push
,repo_hooks_tags
,repo_params
,repo_hash
FROM
 repos r
,stars s
WHERE r.repo_id = s.star_repo_id
  AND s.star_user_id = ?
`
