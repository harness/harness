package database

import (
	"database/sql"
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type PermManager interface {
	// Grant will grant the user read, write and admin persmissions
	// to the specified repository.
	Grant(u *model.User, r *model.Repo, read, write, admin bool) error

	// Revoke will revoke all user permissions to the specified repository.
	Revoke(u *model.User, r *model.Repo) error

	// Read returns true if the specified user has read
	// access to the repository.
	Read(u *model.User, r *model.Repo) (bool, error)

	// Write returns true if the specified user has write
	// access to the repository.
	Write(u *model.User, r *model.Repo) (bool, error)

	// Admin returns true if the specified user is an
	// administrator of the repository.
	Admin(u *model.User, r *model.Repo) (bool, error)

	// Member returns true if the specified user is a
	// collaborator on the repository.
	Member(u *model.User, r *model.Repo) (bool, error)
}

// permManager manages user permissions to access repositories.
type permManager struct {
	*sql.DB
}

// SQL query to retrieve a user's permission to
// access a repository.
const findPermQuery = `
SELECT *
FROM perms
WHERE user_id=?
AND   repo_id=?
LIMIT 1
`

// SQL statement to delete a permission.
const deletePermStmt = `
DELETE FROM perms WHERE user_id=? AND repo_id=?
`

type perm struct {
	ID      int64 `meddler:"perm_id,pk"`
	UserID  int64 `meddler:"user_id"`
	RepoID  int64 `meddler:"repo_id"`
	Read    bool  `meddler:"perm_read"`
	Write   bool  `meddler:"perm_write"`
	Admin   bool  `meddler:"perm_admin"`
	Created int64 `meddler:"perm_created"`
	Updated int64 `meddler:"perm_updated"`
}

// NewManager initiales a new PermManager intended to
// manage user permission and access control.
func NewPermManager(db *sql.DB) PermManager {
	return &permManager{db}
}

// Grant will grant the user read, write and admin persmissions
// to the specified repository.
func (db *permManager) Grant(u *model.User, r *model.Repo, read, write, admin bool) error {
	// attempt to get existing permissions from the database
	perm, err := db.find(u, r)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// if this is a new permission set the user ID,
	// repository ID and created timestamp.
	if perm.ID == 0 {
		perm.UserID = u.ID
		perm.RepoID = r.ID
		perm.Created = time.Now().Unix()
	}

	// set all the permission values
	perm.Read = read
	perm.Write = write
	perm.Admin = admin
	perm.Updated = time.Now().Unix()

	// update the database
	return meddler.Save(db, "perms", perm)
}

// Revoke will revoke all user permissions to the specified repository.
func (db *permManager) Revoke(u *model.User, r *model.Repo) error {
	_, err := db.Exec(deletePermStmt, u.ID, r.ID)
	return err
}

func (db *permManager) Read(u *model.User, r *model.Repo) (bool, error) {
	switch {
	// if the repo is public, grant access.
	case r.Private == false:
		return true, nil
	// if the repo is private and the user is nil, deny access
	case r.Private == true && u == nil:
		return false, nil
	// if the user is a system admin, grant access
	case u.Admin == true:
		return true, nil
	}

	// get the permissions from the database
	perm, err := db.find(u, r)
	return perm.Read, err
}

func (db *permManager) Write(u *model.User, r *model.Repo) (bool, error) {
	switch {
	// if the user is nil, deny access
	case u == nil:
		return false, nil
	// if the user is a system admin, grant access
	case u.Admin == true:
		return true, nil
	}

	// get the permissions from the database
	perm, err := db.find(u, r)
	return perm.Write, err
}

func (db *permManager) Admin(u *model.User, r *model.Repo) (bool, error) {
	switch {
	// if the user is nil, deny access
	case u == nil:
		return false, nil
	// if the user is a system admin, grant access
	case u.Admin == true:
		return true, nil
	}

	// get the permissions from the database
	perm, err := db.find(u, r)
	return perm.Admin, err
}

func (db *permManager) Member(u *model.User, r *model.Repo) (bool, error) {
	switch {
	// if the user is nil, deny access
	case u == nil:
		return false, nil
	case u.ID == r.UserID:
		return true, nil
	}

	// get the permissions from the database
	perm, err := db.find(u, r)
	return perm.Read, err
}

func (db *permManager) find(u *model.User, r *model.Repo) (*perm, error) {
	var dst = perm{}
	var err = meddler.QueryRow(db, &dst, findPermQuery, u.ID, r.ID)
	return &dst, err
}
