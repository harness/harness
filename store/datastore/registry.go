package datastore

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) RegistryFind(repo *model.Repo, addr string) (*model.Registry, error) {
	stmt := sql.Lookup(db.driver, "registry-find-repo-addr")
	data := new(model.Registry)
	err := meddler.QueryRow(db, data, stmt, repo.ID, addr)
	return data, err
}

func (db *datastore) RegistryList(repo *model.Repo) ([]*model.Registry, error) {
	stmt := sql.Lookup(db.driver, "registry-find-repo")
	data := []*model.Registry{}
	err := meddler.QueryAll(db, &data, stmt, repo.ID)
	return data, err
}

func (db *datastore) RegistryCreate(registry *model.Registry) error {
	return meddler.Insert(db, "registry", registry)
}

func (db *datastore) RegistryUpdate(registry *model.Registry) error {
	return meddler.Update(db, "registry", registry)
}

func (db *datastore) RegistryDelete(registry *model.Registry) error {
	stmt := sql.Lookup(db.driver, "registry-delete")
	_, err := db.Exec(stmt, registry.ID)
	return err
}
