// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
