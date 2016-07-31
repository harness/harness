package datastore

import (
	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

func (db *datastore) GetSecretList(repo *model.Repo) ([]*model.RepoSecret, error) {
	var secrets = []*model.RepoSecret{}
	var err = meddler.QueryAll(db, &secrets, rebind(secretListQuery), repo.ID)
	return secrets, err
}

func (db *datastore) GetSecret(repo *model.Repo, name string) (*model.RepoSecret, error) {
	var secret = new(model.RepoSecret)
	var err = meddler.QueryRow(db, secret, rebind(secretNameQuery), repo.ID, name)
	return secret, err
}

func (db *datastore) SetSecret(sec *model.RepoSecret) error {
	var got = new(model.RepoSecret)
	var err = meddler.QueryRow(db, got, rebind(secretNameQuery), sec.RepoID, sec.Name)
	if err == nil && got.ID != 0 {
		sec.ID = got.ID // update existing id
	}
	return meddler.Save(db, secretTable, sec)
}

func (db *datastore) DeleteSecret(sec *model.RepoSecret) error {
	_, err := db.Exec(rebind(secretDeleteStmt), sec.ID)
	return err
}

const secretTable = "secrets"

const secretListQuery = `
SELECT *
FROM secrets
WHERE secret_repo_id = ?
`

const secretNameQuery = `
SELECT *
FROM secrets
WHERE secret_repo_id = ?
  AND secret_name = ?
LIMIT 1;
`

const secretDeleteStmt = `
DELETE FROM secrets
WHERE secret_id = ?
`
