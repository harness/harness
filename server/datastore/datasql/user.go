package datasql

import (
	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type Userstore struct {
	meddler.DB
}

func NewUserstore(db meddler.DB) *Userstore {
	return &Userstore{db}
}

// GetUser retrieves a specific user from the
// datastore for the given ID.
func (db *Repostore) GetUser(id int64) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.Load(db, userTable, usr, id)
	return usr, err
}

// GetUserLogin retrieves a user from the datastore
// for the specified remote and login name.
func (db *Repostore) GetUserLogin(remote, login string) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.QueryRow(db, usr, userLoginQuery)
	return usr, err
}

// GetUserToken retrieves a user from the datastore
// with the specified token.
func (db *Repostore) GetUserToken(token string) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.QueryRow(db, usr, userTokenQuery)
	return usr, err
}

// GetUserList retrieves a list of all users from
// the datastore that are registered in the system.
func (db *Repostore) GetUserList() ([]*model.User, error) {
	var users []*model.User
	var err = meddler.QueryAll(db, &users, userListQuery)
	return users, err
}

// PostUser saves a User in the datastore.
func (db *Repostore) PostUser(user *model.User) error {
	return meddler.Save(db, userTable, user)
}

// PutUser saves a user in the datastore.
func (db *Repostore) PutUser(user *model.User) error {
	return meddler.Save(db, userTable, user)
}

// DelUser removes the user from the datastore.
func (db *Repostore) DelUser(user *model.User) error {
	var _, err = db.Exec(userDeleteStmt, user.ID)
	return err
}

// User table name in database.
const userTable = "users"

// SQL query to retrieve a User by remote login.
const userLoginQuery = `
SELECT *
FROM users
WHERE user_remote=?
AND   user_login=?
LIMIT 1
`

// SQL query to retrieve a User by remote login.
const userTokenQuery = `
SELECT *
FROM users
WHERE user_token=?
LIMIT 1
`

// SQL query to retrieve a list of all users.
const userListQuery = `
SELECT *
FROM users
ORDER BY user_name ASC
`

// SQL statement to delete a User by ID.
const userDeleteStmt = `
DELETE FROM users
WHERE user_id=?
`
