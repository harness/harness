package datastore

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) DeployEnvList(build *model.Build) ([]*model.DeployEnv, error) {
	stmt := sql.Lookup(db.driver, "deploy_envs-find-build")
	list := []*model.DeployEnv{}
	err := meddler.QueryAll(db, &list, stmt, build.ID)
	return list, err
}

func (db *datastore) DeployEnvCreate(deploy_envs []*model.DeployEnv) error {
	for _, deploy_env := range deploy_envs {
		if err := meddler.Insert(db, "deploy_envs", deploy_env); err != nil {
			return err
		}
	}
	return nil
}
