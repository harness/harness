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

func (db *datastore) GetRepo(id int64) (*model.Repo, error) {
	var repo = new(model.Repo)
	var err = meddler.Load(db, repoTable, repo, id)
	return repo, err
}

func (db *datastore) GetRepoName(name string) (*model.Repo, error) {
	var repo = new(model.Repo)
	var err = meddler.QueryRow(db, repo, rebind(repoNameQuery), name)
	return repo, err
}

func (db *datastore) GetRepoCount() (count int, err error) {
	err = db.QueryRow(
		sql.Lookup(db.driver, "count-repos"),
	).Scan(&count)
	return
}

func (db *datastore) CreateRepo(repo *model.Repo) error {
	return meddler.Insert(db, repoTable, repo)
}

func (db *datastore) UpdateRepo(repo *model.Repo) error {
	return meddler.Update(db, repoTable, repo)
}

func (db *datastore) DeleteRepo(repo *model.Repo) error {
	stmt := sql.Lookup(db.driver, "repo-delete")
	_, err := db.Exec(stmt, repo.ID)
	return err
}

func (db *datastore) RepoList(user *model.User) ([]*model.Repo, error) {
	stmt := sql.Lookup(db.driver, "repo-find-user")
	data := []*model.Repo{}
	err := meddler.QueryAll(db, &data, stmt, user.ID)
	return data, err
}

func (db *datastore) RepoListLatest(user *model.User) ([]*model.Feed, error) {
	stmt := sql.Lookup(db.driver, "feed-latest-build")
	data := []*model.Feed{}
	err := meddler.QueryAll(db, &data, stmt, user.ID)
	return data, err
}

func (db *datastore) RepoBatch(repos []*model.Repo) error {
	stmt := sql.Lookup(db.driver, "repo-insert-ignore")
	for _, repo := range repos {
		_, err := db.Exec(stmt,
			repo.UserID,
			repo.Owner,
			repo.Name,
			repo.FullName,
			repo.Avatar,
			repo.Link,
			repo.Clone,
			repo.Branch,
			repo.Timeout,
			repo.IsPrivate,
			repo.IsTrusted,
			repo.IsActive,
			repo.AllowPull,
			repo.AllowPush,
			repo.AllowDeploy,
			repo.AllowTag,
			repo.Hash,
			repo.Kind,
			repo.Config,
			repo.IsGated,
			repo.Visibility,
			repo.Counter,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

const repoTable = "repos"

const repoNameQuery = `
SELECT *
FROM repos
WHERE repo_full_name = ?
LIMIT 1;
`

const repoDeleteStmt = `
DELETE FROM repos
WHERE repo_id = ?
`
