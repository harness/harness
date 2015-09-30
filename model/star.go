package model

import (
	"github.com/CiscoCloud/drone/shared/database"
	"github.com/russross/meddler"
)

type Star struct {
	ID     int64 `meddler:"star_id,pk"`
	RepoID int64 `meddler:"star_repo_id"`
	UserID int64 `meddler:"star_user_id"`
}

func GetStar(db meddler.DB, user *User, repo *Repo) (bool, error) {
	var star = new(Star)
	err := meddler.QueryRow(db, star, database.Rebind(starQuery), user.ID, repo.ID)
	return (err == nil), err
}

func CreateStar(db meddler.DB, user *User, repo *Repo) error {
	var star = &Star{UserID: user.ID, RepoID: repo.ID}
	return meddler.Insert(db, starTable, star)
}

func DeleteStar(db meddler.DB, user *User, repo *Repo) error {
	var _, err = db.Exec(database.Rebind(starDeleteStmt), user.ID, repo.ID)
	return err
}

const starTable = "stars"

const starQuery = `
SELECT *
FROM stars
WHERE star_user_id=?
AND   star_repo_id=?
LIMIT 1
`

const starDeleteStmt = `
DELETE FROM stars
WHERE star_user_id=?
  AND star_repo_id=?
`
