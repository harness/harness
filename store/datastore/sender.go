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

func (db *datastore) SenderFind(repo *model.Repo, login string) (*model.Sender, error) {
	stmt := sql.Lookup(db.driver, "sender-find-repo-login")
	data := new(model.Sender)
	err := meddler.QueryRow(db, data, stmt, repo.ID, login)
	return data, err
}

func (db *datastore) SenderList(repo *model.Repo) ([]*model.Sender, error) {
	stmt := sql.Lookup(db.driver, "sender-find-repo")
	data := []*model.Sender{}
	err := meddler.QueryAll(db, &data, stmt, repo.ID)
	return data, err
}

func (db *datastore) SenderCreate(sender *model.Sender) error {
	return meddler.Insert(db, "senders", sender)
}

func (db *datastore) SenderUpdate(sender *model.Sender) error {
	return meddler.Update(db, "senders", sender)
}

func (db *datastore) SenderDelete(sender *model.Sender) error {
	stmt := sql.Lookup(db.driver, "sender-delete")
	_, err := db.Exec(stmt, sender.ID)
	return err
}
