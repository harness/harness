package database

import (
	"database/sql"
	"time"

	"github.com/drone/drone/server/helper"
	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type PermManager interface {
	// Grant will grant the user read, write and admin persmissions
	// to the specified repository.
	Grant(u *model.User, r *model.Repo, read, write, admin bool) error

	// Revoke will revoke all user permissions to the specified repository.
	Revoke(u *model.User, r *model.Repo) error

	// Find returns the user's permission to access the specified repository.
	Find(u *model.User, r *model.Repo) *model.Perm

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
	_, err := db.Exec(helper.Rebind(deletePermStmt), u.ID, r.ID)
	return err
}

func (db *permManager) Find(u *model.User, r *model.Repo) *model.Perm {
	// if the user is a gues they should only be granted
	// read access to public repositories.
	switch {
	case u == nil && r.Private:
		return &model.Perm{
			Guest: true,
			Read:  false,
			Write: false,
			Admin: false}
	case u == nil && !r.Private:
		return &model.Perm{
			Guest: true,
			Read:  true,
			Write: false,
			Admin: false}
	}

	// if the user is authenticated we'll retireive the
	// permission details from the database.
	perm, err := db.find(u, r)
	if err != nil && perm.ID != 0 {
		return perm
	}

	switch {
	// if the user is a system admin grant super access.
	case u.Admin == true:
		perm.Read = true
		perm.Write = true
		perm.Admin = true
		perm.Guest = true

	// if the repo is public, grant read access only.
	case r.Private == false:
		perm.Read = true
		perm.Guest = true
	}

	return perm
}

func (db *permManager) Read(u *model.User, r *model.Repo) (bool, error) {
	return db.Find(u, r).Read, nil
}

func (db *permManager) Write(u *model.User, r *model.Repo) (bool, error) {
	return db.Find(u, r).Write, nil
}

func (db *permManager) Admin(u *model.User, r *model.Repo) (bool, error) {
	return db.Find(u, r).Admin, nil
}

func (db *permManager) Member(u *model.User, r *model.Repo) (bool, error) {
	perm := db.Find(u, r)
	return perm.Read && !perm.Guest, nil
}

func (db *permManager) find(u *model.User, r *model.Repo) (*model.Perm, error) {
	var dst = model.Perm{}
	var err = meddler.QueryRow(db, &dst, helper.Rebind(findPermQuery), u.ID, r.ID)
	return &dst, err
}
