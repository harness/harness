package datastore

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) ConfigLoad(id int64) (*model.Config, error) {
	stmt := sql.Lookup(db.driver, "config-find-repo-id")
	conf := new(model.Config)
	err := meddler.QueryRow(db, conf, stmt, id)
	return conf, err
}

func (db *datastore) ConfigFind(repo *model.Repo, hash string) (*model.Config, error) {
	stmt := sql.Lookup(db.driver, "config-find-repo-hash")
	conf := new(model.Config)
	err := meddler.QueryRow(db, conf, stmt, repo.ID, hash)
	return conf, err
}

func (db *datastore) ConfigUpdate(config *model.Config) error {
	return meddler.Update(db, "config", config)
}

func (db *datastore) ConfigInsert(config *model.Config) error {
	return meddler.Insert(db, "config", config)
}
