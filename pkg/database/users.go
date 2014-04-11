package database

import (
	"time"

	. "github.com/drone/drone/pkg/model"
	"github.com/russross/meddler"
)

// Name of the User table in the database
const userTable = "users"

// SQL Queries to retrieve a user by their unique database key
const userFindIdStmt = `
SELECT id, email, password, token, name, gravatar, created, updated, admin,
github_login, github_token, github_write_token, bitbucket_login, bitbucket_token, bitbucket_secret,
gitlab_token
FROM users WHERE id = ?
`

// SQL Queries to retrieve a user by their email address
const userFindEmailStmt = `
SELECT id, email, password, token, name, gravatar, created, updated, admin,
github_login, github_token, github_write_token, bitbucket_login, bitbucket_token, bitbucket_secret,
gitlab_token
FROM users WHERE email = ?
`

// SQL Queries to retrieve a list of all users
const userStmt = `
SELECT id, email, password, token, name, gravatar, created, updated, admin,
github_login, github_token, github_write_token, bitbucket_login, bitbucket_token, bitbucket_secret,
gitlab_token
FROM users
ORDER BY name ASC
`

// Returns the User with the given ID.
func GetUser(id int64) (*User, error) {
	user := User{}
	err := meddler.QueryRow(db, &user, userFindIdStmt, id)
	return &user, err
}

// Returns the User with the given email address.
func GetUserEmail(email string) (*User, error) {
	user := User{}
	err := meddler.QueryRow(db, &user, userFindEmailStmt, email)
	return &user, err
}

// Returns the User Password Hash for the given
// email address.
func GetPassEmail(email string) ([]byte, error) {
	user, err := GetUserEmail(email)
	if err != nil {
		return nil, err
	}

	return []byte(user.Password), nil
}

// Saves the User account.
func SaveUser(user *User) error {
	if user.ID == 0 {
		user.Created = time.Now().UTC()
	}
	user.Updated = time.Now().UTC()
	return meddler.Save(db, userTable, user)
}

// Deletes an existing User account.
func DeleteUser(id int64) error {
	db.Exec("DELETE FROM members WHERE user_id = ?", id)
	db.Exec("DELETE FROM users WHERE id = ?", id)
	// TODO delete all projects
	return nil
}

// Returns a list of all Users.
func ListUsers() ([]*User, error) {
	var users []*User
	err := meddler.QueryAll(db, &users, userStmt)
	return users, err
}

// Returns a list of Users within the specified
// range (for pagination purposes).
func ListUsersRange(limit, offset int) ([]*User, error) {
	var users []*User
	err := meddler.QueryAll(db, &users, userStmt)
	return users, err
}
