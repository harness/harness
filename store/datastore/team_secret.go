package datastore

import (
	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

func (db *datastore) GetTeamSecretList(team string) ([]*model.TeamSecret, error) {
	var secrets = []*model.TeamSecret{}
	var err = meddler.QueryAll(db, &secrets, rebind(teamSecretListQuery), team)
	return secrets, err
}

func (db *datastore) GetTeamSecret(team, name string) (*model.TeamSecret, error) {
	var secret = new(model.TeamSecret)
	var err = meddler.QueryRow(db, secret, rebind(teamSecretNameQuery), team, name)
	return secret, err
}

func (db *datastore) SetTeamSecret(sec *model.TeamSecret) error {
	var got = new(model.TeamSecret)
	var err = meddler.QueryRow(db, got, rebind(teamSecretNameQuery), sec.Key, sec.Name)
	if err == nil && got.ID != 0 {
		sec.ID = got.ID // update existing id
	}
	return meddler.Save(db, teamSecretTable, sec)
}

func (db *datastore) DeleteTeamSecret(sec *model.TeamSecret) error {
	_, err := db.Exec(rebind(teamSecretDeleteStmt), sec.ID)
	return err
}

const teamSecretTable = "team_secrets"

const teamSecretListQuery = `
SELECT *
FROM team_secrets
WHERE team_secret_key = ?
`

const teamSecretNameQuery = `
SELECT *
FROM team_secrets
WHERE team_secret_key = ?
  AND team_secret_name = ?
LIMIT 1;
`

const teamSecretDeleteStmt = `
DELETE FROM team_secrets
WHERE team_secret_id = ?
`
