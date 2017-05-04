package datastore

import (
	"fmt"

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

func (db *datastore) GetRepoListOf(listof []*model.RepoLite) ([]*model.Repo, error) {
	var (
		repos []*model.Repo
		args  []interface{}
		stmt  string
		err   error
	)
	switch meddler.Default {
	case meddler.PostgreSQL:
		stmt, args = toListPostgres(listof)
	default:
		stmt, args = toList(listof)
	}
	if len(args) > 0 {
		err = meddler.QueryAll(db, &repos, fmt.Sprintf(repoListOfQuery, stmt), args...)
	}
	return repos, err
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
	var _, err = db.Exec(rebind(repoDeleteStmt), repo.ID)
	return err
}

const repoTable = "repos"

const repoNameQuery = `
SELECT *
FROM repos
WHERE repo_full_name = ?
LIMIT 1;
`

const repoListOfQuery = `
SELECT *
FROM repos
WHERE repo_full_name IN (%s)
ORDER BY repo_name
`

const repoCountQuery = `
SELECT COUNT(*) FROM repos
`

const repoDeleteStmt = `
DELETE FROM repos
WHERE repo_id = ?
`
