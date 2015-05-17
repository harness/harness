package builtin

import (
	common "github.com/drone/drone/pkg/types"
	"github.com/russross/meddler"
)

type Starstore struct {
	meddler.DB
}

func NewStarstore(db meddler.DB) *Starstore {
	return &Starstore{db}
}

// Starred returns true if the user starred
// the given repository.
func (db *Starstore) Starred(user *common.User, repo *common.Repo) (bool, error) {
	var star = new(star)
	err := meddler.QueryRow(db, star, rebind(starQuery), user.ID, repo.ID)
	return (err == nil), err
}

// AddStar inserts a starred repo / user in the datastore.
func (db *Starstore) AddStar(user *common.User, repo *common.Repo) error {
	var star = &star{UserID: user.ID, RepoID: repo.ID}
	return meddler.Insert(db, starTable, star)
}

// DelStar removes starred repo / user from the datastore.
func (db *Starstore) DelStar(user *common.User, repo *common.Repo) error {
	var _, err = db.Exec(rebind(starDeleteStmt), user.ID, repo.ID)
	return err
}

type star struct {
	ID     int64 `meddler:"star_id,pk"`
	UserID int64 `meddler:"user_id"`
	RepoID int64 `meddler:"repo_id"`
}

// Stars table name in database.
const starTable = "stars"

// SQL query to retrieve a user's stars to
// access a repository.
const starQuery = `
SELECT *
FROM stars
WHERE user_id=?
AND   repo_id=?
LIMIT 1
`

// SQL statement to delete a star by ID.
const starDeleteStmt = `
DELETE FROM stars
WHERE user_id=?
  AND repo_id=?
`
