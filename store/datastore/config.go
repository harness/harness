package datastore

import (
	gosql "database/sql"

	"github.com/drone/drone/model"
	"github.com/drone/drone/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) ConfigLoad(id int64) (*model.Config, error) {
	stmt := sql.Lookup(db.driver, "config-find-id")
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

func (db *datastore) ConfigFindApproved(config *model.Config) (bool, error) {
	var dest int64
	stmt := sql.Lookup(db.driver, "config-find-approved")
	err := db.DB.QueryRow(stmt, config.RepoID, config.ID).Scan(&dest)
	if err == gosql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (db *datastore) ConfigCreate(config *model.Config) error {
	return meddler.Insert(db, "config", config)
}
