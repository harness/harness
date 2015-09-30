package model

import (
	"github.com/CiscoCloud/drone/shared/database"
	"github.com/russross/meddler"
)

type Key struct {
	ID      int64  `json:"-"       meddler:"key_id,pk"`
	RepoID  int64  `json:"-"       meddler:"key_repo_id"`
	Public  string `json:"public"  meddler:"key_public"`
	Private string `json:"private" meddler:"key_private"`
}

func GetKey(db meddler.DB, repo *Repo) (*Key, error) {
	var key = new(Key)
	var err = meddler.QueryRow(db, key, database.Rebind(keyQuery), repo.ID)
	return key, err
}

func CreateKey(db meddler.DB, key *Key) error {
	return meddler.Save(db, keyTable, key)
}

func UpdateKey(db meddler.DB, key *Key) error {
	return meddler.Save(db, keyTable, key)
}

func DeleteKey(db meddler.DB, repo *Repo) error {
	var _, err = db.Exec(database.Rebind(keyDeleteStmt), repo.ID)
	return err
}

const keyTable = "keys"

const keyQuery = `
SELECT *
FROM keys
WHERE key_repo_id=?
LIMIT 1
`

const keyDeleteStmt = `
DELETE FROM keys
WHERE key_repo_id=?
`
