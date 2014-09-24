package database

import (
	"database/sql"
	"time"

	"github.com/drone/drone/server/helper"
	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type UserManager interface {
	// Find finds the User by ID.
	Find(id int64) (*model.User, error)

	// FindLogin finds the User by remote login.
	FindLogin(remote, login string) (*model.User, error)

	// FindToken finds the User by token.
	FindToken(token string) (*model.User, error)

	// List finds all registered users of the system.
	List() ([]*model.User, error)

	// Insert persists the User to the datastore.
	Insert(user *model.User) error

	// Update persists changes to the User to the datastore.
	Update(user *model.User) error

	// Delete removes the User from the datastore.
	Delete(user *model.User) error

	// Exist returns true if Users exist in the system.
	Exist() bool
}

// userManager manages a list of users in a SQL database.
type userManager struct {
	*sql.DB
}

// SQL query to retrieve a User by remote login.
const findUserLoginQuery = `
SELECT *
FROM users
WHERE user_remote=?
AND   user_login=?
LIMIT 1
`

// SQL query to retrieve a User by remote login.
const findUserTokenQuery = `
SELECT *
FROM users
WHERE user_token=?
LIMIT 1
`

// SQL query to retrieve a list of all users.
const listUserQuery = `
SELECT *
FROM users
ORDER BY user_name ASC
`

// SQL statement to delete a User by ID.
const deleteUserStmt = `
DELETE FROM users WHERE user_id=?
`

// SQL statement to check if users exist.
const confirmUserStmt = `
select 1 from users limit 1
`

// NewUserManager initiales a new UserManager intended to
// manage and persist commits.
func NewUserManager(db *sql.DB) UserManager {
	return &userManager{db}
}

func (db *userManager) Find(id int64) (*model.User, error) {
	dst := model.User{}
	err := meddler.Load(db, "users", &dst, id)
	return &dst, err
}

func (db *userManager) FindLogin(remote, login string) (*model.User, error) {
	dst := model.User{}
	err := meddler.QueryRow(db, &dst, helper.Rebind(findUserLoginQuery), remote, login)
	return &dst, err
}

func (db *userManager) FindToken(token string) (*model.User, error) {
	dst := model.User{}
	err := meddler.QueryRow(db, &dst, helper.Rebind(findUserTokenQuery), token)
	return &dst, err
}

func (db *userManager) List() ([]*model.User, error) {
	var dst []*model.User
	err := meddler.QueryAll(db, &dst, helper.Rebind(listUserQuery))
	return dst, err
}

func (db *userManager) Insert(user *model.User) error {
	user.Created = time.Now().Unix()
	user.Updated = time.Now().Unix()
	return meddler.Insert(db, "users", user)
}

func (db *userManager) Update(user *model.User) error {
	user.Updated = time.Now().Unix()
	return meddler.Update(db, "users", user)
}

func (db *userManager) Delete(user *model.User) error {
	_, err := db.Exec(helper.Rebind(deleteUserStmt), user.ID)
	return err
}

func (db *userManager) Exist() bool {
	row := db.QueryRow(helper.Rebind(confirmUserStmt))
	var result int
	row.Scan(&result)
	return result == 1
}
