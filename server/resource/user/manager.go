package user

import (
	"database/sql"
	"time"

	"github.com/russross/meddler"
)

type UserManager interface {
	// Find finds the User by ID.
	Find(id int64) (*User, error)

	// FindLogin finds the User by remote login.
	FindLogin(remote, login string) (*User, error)

	// FindToken finds the User by token.
	FindToken(token string) (*User, error)

	// List finds all registered users of the system.
	List() ([]*User, error)

	// Insert persists the User to the datastore.
	Insert(user *User) error

	// Update persists changes to the User to the datastore.
	Update(user *User) error

	// Delete removes the User from the datastore.
	Delete(user *User) error
}

// userManager manages a list of users in a SQL database.
type userManager struct {
	*sql.DB
}

// SQL query to retrieve a User by remote login.
const findLoginQuery = `
SELECT *
FROM users
WHERE user_remote=?
AND   user_login=?
LIMIT 1
`

// SQL query to retrieve a User by remote login.
const findTokenQuery = `
SELECT *
FROM users
WHERE user_token=?
LIMIT 1
`

// SQL query to retrieve a list of all users.
const listQuery = `
SELECT *
FROM users
ORDER BY user_name ASC
`

// SQL statement to delete a User by ID.
const deleteStmt = `
DELETE FROM users WHERE user_id=?
`

// NewManager initiales a new UserManager intended to
// manage and persist commits.
func NewManager(db *sql.DB) UserManager {
	return &userManager{db}
}

func (db *userManager) Find(id int64) (*User, error) {
	dst := User{}
	err := meddler.Load(db, "users", &dst, id)
	return &dst, err
}

func (db *userManager) FindLogin(remote, login string) (*User, error) {
	dst := User{}
	err := meddler.QueryRow(db, &dst, findLoginQuery, remote, login)
	return &dst, err
}

func (db *userManager) FindToken(token string) (*User, error) {
	dst := User{}
	err := meddler.QueryRow(db, &dst, findTokenQuery, token)
	return &dst, err
}

func (db *userManager) List() ([]*User, error) {
	var dst []*User
	err := meddler.QueryAll(db, &dst, listQuery)
	return dst, err
}

func (db *userManager) Insert(user *User) error {
	user.Created = time.Now().Unix()
	user.Updated = time.Now().Unix()
	return meddler.Insert(db, "users", user)
}

func (db *userManager) Update(user *User) error {
	user.Updated = time.Now().Unix()
	return meddler.Update(db, "users", user)
}

func (db *userManager) Delete(user *User) error {
	_, err := db.Exec(deleteStmt, user.ID)
	return err
}
