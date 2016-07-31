package datastore

import (
	"github.com/drone/drone/model"
)

func (db *datastore) GetMergedSecretList(repo *model.Repo) ([]*model.Secret, error) {
	var (
		secrets []*model.Secret
	)

	repoSecs, err := db.GetSecretList(repo)

	if err != nil {
		return nil, err
	}

	for _, secret := range repoSecs {
		secrets = append(secrets, secret.Secret())
	}

	teamSecs, err := db.GetTeamSecretList(repo.Owner)

	if err != nil {
		return nil, err
	}

	for _, secret := range teamSecs {
		secrets = append(secrets, secret.Secret())
	}

	return secrets, nil
}
