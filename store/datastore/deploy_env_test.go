package datastore

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestDeployEnvList(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from deploy_envs")
		s.Close()
	}()

	err := s.DeployEnvCreate([]*model.DeployEnv{
		{
			BuildID: 1,
			Name:    "test",
		},
		{
			BuildID: 1,
			Name:    "dev",
		},
		{
			BuildID: 1,
			Name:    "prod",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: insert deploy_envs: %s", err)
		return
	}
	deploy_envs, err := s.DeployEnvList(&model.Build{ID: 1})
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := len(deploy_envs), 3; got != want {
		t.Errorf("Want %d deploy_envs, got %d", want, got)
	}
}
