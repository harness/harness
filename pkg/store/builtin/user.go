package builtin

import (
	"database/sql"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/russross/meddler"
	"github.com/drone/drone/pkg/types"
)

type Userstore struct {
	*sql.DB
}

func NewUserstore(db *sql.DB) *Userstore {
	return &Userstore{db}
}

// User returns a user by user ID.
func (db *Userstore) User(id int64) (*types.User, error) {
	return getUser(db, rebind(stmtUserSelect), id)
}

// UserLogin returns a user by user login.
func (db *Userstore) UserLogin(login string) (*types.User, error) {
	return getUser(db, rebind(stmtUserSelectUserLogin), login)
}

// UserList returns a list of all registered users.
func (db *Userstore) UserList() ([]*types.User, error) {
	return getUsers(db, rebind(stmtUserSelectList))
}

// UserFeed retrieves a digest of recent builds
// from the datastore accessible to the specified user.
func (db *Userstore) UserFeed(user *types.User, limit, offset int) ([]*types.RepoCommit, error) {
	var builds []*types.RepoCommit
	var err = meddler.QueryAll(db, &builds, rebind(userFeedQuery), user.ID, limit, offset)
	return builds, err
}

// UserCount returns a count of all registered users.
func (db *Userstore) UserCount() (int, error) {
	var count int
	err := db.QueryRow(stmtUserSelectCount).Scan(&count)
	return count, err
}

// AddUser inserts a new user into the datastore.
// If the user login already exists an error is returned.
func (db *Userstore) AddUser(user *types.User) error {
	user.Created = time.Now().UTC().Unix()
	user.Updated = time.Now().UTC().Unix()
	return createUser(db, rebind(stmtUserInsert), user)
}

// SetUser updates an existing user.
func (db *Userstore) SetUser(user *types.User) error {
	user.Updated = time.Now().UTC().Unix()
	return updateUser(db, rebind(stmtUserUpdate), user)
}

// DelUser removes the user from the datastore.
func (db *Userstore) DelUser(user *types.User) error {
	var _, err = db.Exec(rebind(stmtUserDelete), user.ID)
	return err
}

// SQL query to retrieve a build feed for the given
// user account.
const userFeedQuery = `
SELECT
 r.repo_id
,r.repo_owner
,r.repo_name
,r.repo_full_name
,c.commit_seq
,c.commit_state
,c.commit_started
,c.commit_finished
FROM
 commits c
,repos r
,stars s
WHERE c.repo_id = r.repo_id
  AND r.repo_id = s.repo_id
  AND s.user_id = ?
ORDER BY c.commit_seq DESC
LIMIT ? OFFSET ?
`
