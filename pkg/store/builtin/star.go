package builtin

import (
	"database/sql"

	"github.com/drone/drone/pkg/types"
)

type Starstore struct {
	*sql.DB
}

func NewStarstore(db *sql.DB) *Starstore {
	return &Starstore{db}
}

// Starred returns true if the user starred
// the given repository.
func (db *Starstore) Starred(user *types.User, repo *types.Repo) (bool, error) {
	_, err := getStar(db, rebind(stmtStarSelectStarUserRepo), user.ID, repo.ID)
	return (err == nil), err
}

// AddStar inserts a starred repo / user in the datastore.
func (db *Starstore) AddStar(user *types.User, repo *types.Repo) error {
	var star = &Star{UserID: user.ID, RepoID: repo.ID}
	return createStar(db, rebind(stmtStarInsert), star)
}

// DelStar removes starred repo / user from the datastore.
func (db *Starstore) DelStar(user *types.User, repo *types.Repo) error {
	var _, err = db.Exec(rebind(stmtStartDeleteUserRepo), user.ID, repo.ID)
	return err
}

type Star struct {
	ID     int64
	UserID int64 `sql:"unique:ux_star_user_repo"`
	RepoID int64 `sql:"unique:ux_star_user_repo"`
}

// SQL statement to delete a star by ID.
const stmtStartDeleteUserRepo = `
DELETE FROM stars
WHERE star_user_id=?
  AND star_repo_id=?
`
