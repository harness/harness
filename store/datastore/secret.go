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

func (db *datastore) SecretFind(repo *model.Repo, name string) (*model.Secret, error) {
	stmt := sql.Lookup(db.driver, "secret-find-repo-name")
	data := new(model.Secret)
	err := meddler.QueryRow(db, data, stmt, repo.ID, name)
	return data, err
}

func (db *datastore) SecretList(repo *model.Repo) ([]*model.Secret, error) {
	stmt := sql.Lookup(db.driver, "secret-find-repo")
	data := []*model.Secret{}
	err := meddler.QueryAll(db, &data, stmt, repo.ID)
	return data, err
}

func (db *datastore) SecretCreate(secret *model.Secret) error {
	return meddler.Insert(db, "secrets", secret)
}

func (db *datastore) SecretUpdate(secret *model.Secret) error {
	return meddler.Update(db, "secrets", secret)
}

func (db *datastore) SecretDelete(secret *model.Secret) error {
	stmt := sql.Lookup(db.driver, "secret-delete")
	_, err := db.Exec(stmt, secret.ID)
	return err
}
