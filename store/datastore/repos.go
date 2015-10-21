package datastore

import (
	"database/sql"
	"strings"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

type repostore struct {
	*sql.DB
}

func (db *repostore) Get(id int64) (*model.Repo, error) {
	var repo = new(model.Repo)
	var err = meddler.Load(db, repoTable, repo, id)
	return repo, err
}

func (db *repostore) GetName(name string) (*model.Repo, error) {
	var repo = new(model.Repo)
	var err = meddler.QueryRow(db, repo, rebind(repoNameQuery), name)
	return repo, err
}

func (db *repostore) GetListOf(listof []*model.RepoLite) ([]*model.Repo, error) {
	var repos = []*model.Repo{}
	var size = len(listof)
	if size > 999 {
		size = 999
		listof = listof[:999]
	}
	var qs = make([]string, size, size)
	var in = make([]interface{}, size, size)
	for i, repo := range listof {
		qs[i] = "?"
		in[i] = repo.FullName
	}
	var stmt = "SELECT * FROM repos WHERE repo_full_name IN (" + strings.Join(qs, ",") + ") ORDER BY repo_name"
	var err = meddler.QueryAll(db, &repos, rebind(stmt), in...)
	return repos, err
}

func (db *repostore) Count() (int, error) {
	var count int
	var err = db.QueryRow(rebind(repoCountQuery)).Scan(&count)
	return count, err
}

func (db *repostore) Create(repo *model.Repo) error {
	return meddler.Insert(db, repoTable, repo)
}

func (db *repostore) Update(repo *model.Repo) error {
	return meddler.Update(db, repoTable, repo)
}

func (db *repostore) Delete(repo *model.Repo) error {
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

const repoListQuery = `
SELECT *
FROM repos
WHERE repo_id IN (
	SELECT DISTINCT build_repo_id
	FROM builds
	WHERE build_author = ?
)
ORDER BY repo_full_name
`

const repoCountQuery = `
SELECT COUNT(*) FROM repos
`

const repoDeleteStmt = `
DELETE FROM repos
WHERE repo_id = ?
`
