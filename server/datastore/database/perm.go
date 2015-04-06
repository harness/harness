package database

import (
	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type Permstore struct {
	meddler.DB
}

func NewPermstore(db meddler.DB) *Permstore {
	return &Permstore{db}
}

// GetPerm retrieves the User's permission from
// the datastore for the given repository.
func (db *Permstore) GetPerm(user *model.User, repo *model.Repo) (*model.Perm, error) {
	var perm = new(model.Perm)
	var err = meddler.QueryRow(db, perm, rebind(permQuery), user.ID, repo.ID)
	return perm, err
}

// PostPerm saves permission in the datastore.
func (db *Permstore) PostPerm(perm *model.Perm) error {
	var _perm = new(model.Perm)
	meddler.QueryRow(db, _perm, rebind(permQuery), perm.UserID, perm.RepoID)
	if _perm.ID != 0 {
		perm.ID = _perm.ID
	}
	return meddler.Save(db, permTable, perm)
}

// PutPerm saves permission in the datastore.
func (db *Permstore) PutPerm(perm *model.Perm) error {
	return meddler.Save(db, permTable, perm)
}

// DelPerm removes permission from the datastore.
func (db *Permstore) DelPerm(perm *model.Perm) error {
	var _, err = db.Exec(rebind(permDeleteStmt), perm.ID)
	return err
}

// Permission table name in database.
const permTable = "perms"

// SQL query to retrieve a user's permission to
// access a repository.
const permQuery = `
SELECT *
FROM perms
WHERE user_id=?
AND   repo_id=?
LIMIT 1
`

// SQL statement to delete a User by ID.
const permDeleteStmt = `
DELETE FROM perms
WHERE perm_id=?
`
