package builtin

import (
	"time"

	common "github.com/drone/drone/pkg/types"
	"github.com/russross/meddler"
)

type Userstore struct {
	meddler.DB
}

func NewUserstore(db meddler.DB) *Userstore {
	return &Userstore{db}
}

// User returns a user by user ID.
func (db *Userstore) User(id int64) (*common.User, error) {
	var usr = new(common.User)
	var err = meddler.Load(db, userTable, usr, id)
	return usr, err
}

// UserLogin returns a user by user login.
func (db *Userstore) UserLogin(login string) (*common.User, error) {
	var usr = new(common.User)
	var err = meddler.QueryRow(db, usr, rebind(userLoginQuery), login)
	return usr, err
}

// UserList returns a list of all registered users.
func (db *Userstore) UserList() ([]*common.User, error) {
	var users []*common.User
	var err = meddler.QueryAll(db, &users, rebind(userListQuery))
	return users, err
}

// UserFeed retrieves a digest of recent builds
// from the datastore accessible to the specified user.
func (db *Userstore) UserFeed(user *common.User, limit, offset int) ([]*common.RepoCommit, error) {
	var builds []*common.RepoCommit
	var err = meddler.QueryAll(db, &builds, rebind(userFeedQuery), user.ID, limit, offset)
	return builds, err
}

// UserCount returns a count of all registered users.
func (db *Userstore) UserCount() (int, error) {
	var count = struct{ Count int }{}
	var err = meddler.QueryRow(db, &count, rebind(userCountQuery))
	return count.Count, err
}

// AddUser inserts a new user into the datastore.
// If the user login already exists an error is returned.
func (db *Userstore) AddUser(user *common.User) error {
	user.Created = time.Now().UTC().Unix()
	user.Updated = time.Now().UTC().Unix()
	return meddler.Insert(db, userTable, user)
}

// SetUser updates an existing user.
func (db *Userstore) SetUser(user *common.User) error {
	user.Updated = time.Now().UTC().Unix()
	return meddler.Update(db, userTable, user)
}

// DelUser removes the user from the datastore.
func (db *Userstore) DelUser(user *common.User) error {
	var _, err = db.Exec(rebind(userDeleteStmt), user.ID)
	return err
}

// User table name in database.
const userTable = "users"

// SQL query to retrieve a User by remote login.
const userLoginQuery = `
SELECT *
FROM users
WHERE user_login=?
LIMIT 1
`

// SQL query to retrieve a list of all users.
const userListQuery = `
SELECT *
FROM users
ORDER BY user_name ASC
`

// SQL query to retrieve a list of all users.
const userCountQuery = `
SELECT count(1) as "Count"
FROM users
`

// SQL statement to delete a User by ID.
const userDeleteStmt = `
DELETE FROM users
WHERE user_id=?
`

// SQL query to retrieve a build feed for the given
// user account.
const userFeedQuery = `
SELECT
 r.repo_id
,r.repo_owner
,r.repo_name
,r.repo_slug
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
