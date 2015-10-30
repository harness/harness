package datastore

import (
	"database/sql"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

type keystore struct {
	*sql.DB
}

func (db *keystore) Get(repo *model.Repo) (*model.Key, error) {
	var key = new(model.Key)
	var err = meddler.QueryRow(db, key, rebind(keyQuery), repo.ID)
	return key, err
}

func (db *keystore) Create(key *model.Key) error {
	return meddler.Save(db, keyTable, key)
}

func (db *keystore) Update(key *model.Key) error {
	return meddler.Save(db, keyTable, key)
}

func (db *keystore) Delete(key *model.Key) error {
	var _, err = db.Exec(rebind(keyDeleteStmt), key.ID)
	return err
}

const keyTable = "keys"

const keyQuery = "SELECT * FROM `keys` WHERE key_repo_id=? LIMIT 1"

const keyDeleteStmt = "DELETE FROM `keys` WHERE key_id=?"
