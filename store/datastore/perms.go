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

func (db *datastore) PermFind(user *model.User, repo *model.Repo) (*model.Perm, error) {
	stmt := sql.Lookup(db.driver, "perms-find-user-repo")
	data := new(model.Perm)
	err := meddler.QueryRow(db, data, stmt, user.ID, repo.ID)
	return data, err
}

func (db *datastore) PermUpsert(perm *model.Perm) error {
	stmt := sql.Lookup(db.driver, "perms-insert-replace-lookup")
	_, err := db.Exec(stmt,
		perm.UserID,
		perm.Repo,
		perm.Pull,
		perm.Push,
		perm.Admin,
		perm.Synced,
	)
	return err
}

func (db *datastore) PermBatch(perms []*model.Perm) (err error) {
	for _, perm := range perms {
		err = db.PermUpsert(perm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *datastore) PermDelete(perm *model.Perm) error {
	stmt := sql.Lookup(db.driver, "perms-delete-user-repo")
	_, err := db.Exec(stmt, perm.UserID, perm.RepoID)
	return err
}

func (db *datastore) PermFlush(user *model.User, before int64) error {
	stmt := sql.Lookup(db.driver, "perms-delete-user-date")
	_, err := db.Exec(stmt, user.ID, before)
	return err
}
