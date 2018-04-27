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

func (db *datastore) GetUser(id int64) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.Load(db, "users", usr, id)
	return usr, err
}

func (db *datastore) GetUserLogin(login string) (*model.User, error) {
	stmt := sql.Lookup(db.driver, "user-find-login")
	data := new(model.User)
	err := meddler.QueryRow(db, data, stmt, login)
	return data, err
}

func (db *datastore) GetUserList() ([]*model.User, error) {
	stmt := sql.Lookup(db.driver, "user-find")
	data := []*model.User{}
	err := meddler.QueryAll(db, &data, stmt)
	return data, err
}

func (db *datastore) GetUserCount() (count int, err error) {
	err = db.QueryRow(
		sql.Lookup(db.driver, "count-users"),
	).Scan(&count)
	return
}

func (db *datastore) CreateUser(user *model.User) error {
	return meddler.Insert(db, "users", user)
}

func (db *datastore) UpdateUser(user *model.User) error {
	return meddler.Update(db, "users", user)
}

func (db *datastore) DeleteUser(user *model.User) error {
	stmt := sql.Lookup(db.driver, "user-delete")
	_, err := db.Exec(stmt, user.ID)
	return err
}

func (db *datastore) UserFeed(user *model.User) ([]*model.Feed, error) {
	stmt := sql.Lookup(db.driver, "feed")
	data := []*model.Feed{}
	err := meddler.QueryAll(db, &data, stmt, user.ID)
	return data, err
}
