package database

import (
	"time"

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
func (db *Userstore) GetUser(id int64) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.Load(db, userTable, usr, id)
	return usr, err
}

// GetUserLogin retrieves a user from the datastore
// for the specified remote and login name.
func (db *Userstore) GetUserLogin(remote, login string) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.QueryRow(db, usr, rebind(userLoginQuery), remote, login)
	return usr, err
}

// GetUserToken retrieves a user from the datastore
// with the specified token.
func (db *Userstore) GetUserToken(token string) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.QueryRow(db, usr, rebind(userTokenQuery), token)
	return usr, err
}

// GetUserList retrieves a list of all users from
// the datastore that are registered in the system.
func (db *Userstore) GetUserList() ([]*model.User, error) {
	var users []*model.User
	var err = meddler.QueryAll(db, &users, rebind(userListQuery))
	return users, err
}

// PostUser saves a User in the datastore.
func (db *Userstore) PostUser(user *model.User) error {
	user.Created = time.Now().UTC().Unix()
	user.Updated = time.Now().UTC().Unix()
	return meddler.Insert(db, userTable, user)
}

// PutUser saves a user in the datastore.
func (db *Userstore) PutUser(user *model.User) error {
	user.Updated = time.Now().UTC().Unix()
	return meddler.Update(db, userTable, user)
}

// DelUser removes the user from the datastore.
func (db *Userstore) DelUser(user *model.User) error {
	var _, err = db.Exec(rebind(userDeleteStmt), user.ID)
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
