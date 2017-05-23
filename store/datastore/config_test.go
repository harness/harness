package datastore

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestConfig(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from config")
		s.Close()
	}()

	var (
		data = "pipeline: [ { image: golang, commands: [ go build, go test ] } ]"
		hash = "8d8647c9aa90d893bfb79dddbe901f03e258588121e5202632f8ae5738590b26"
	)

	if err := s.ConfigCreate(
		&model.Config{
			RepoID: 2,
			Data:   data,
			Hash:   hash,
		},
	); err != nil {
		t.Errorf("Unexpected error: insert config: %s", err)
		return
	}

	config, err := s.ConfigFind(&model.Repo{ID: 2}, hash)
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := config.ID, int64(1); got != want {
		t.Errorf("Want config id %d, got %d", want, got)
	}
	if got, want := config.RepoID, int64(2); got != want {
		t.Errorf("Want config repo id %d, got %d", want, got)
	}
	if got, want := config.Data, data; got != want {
		t.Errorf("Want config data %s, got %s", want, got)
	}
	if got, want := config.Hash, hash; got != want {
		t.Errorf("Want config hash %s, got %s", want, got)
	}

	loaded, err := s.ConfigLoad(config.ID)
	if err != nil {
		t.Errorf("Want config by id, got error %q", err)
		return
	}
	if got, want := loaded.ID, config.ID; got != want {
		t.Errorf("Want config by id %d, got %d", want, got)
	}
}

func TestConfigApproved(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from config")
		s.Exec("delete from builds")
		s.Exec("delete from repos")
		s.Close()
	}()

	repo := &model.Repo{
		UserID:   1,
		FullName: "bradrydzewski/drone",
		Owner:    "bradrydzewski",
		Name:     "drone",
	}
	s.CreateRepo(repo)

	var (
		data = "pipeline: [ { image: golang, commands: [ go build, go test ] } ]"
		hash = "8d8647c9aa90d893bfb79dddbe901f03e258588121e5202632f8ae5738590b26"
		conf = &model.Config{
			RepoID: repo.ID,
			Data:   data,
			Hash:   hash,
		}
	)

	if err := s.ConfigCreate(conf); err != nil {
		t.Errorf("Unexpected error: insert config: %s", err)
		return
	}
	s.CreateBuild(&model.Build{
		RepoID:   repo.ID,
		ConfigID: conf.ID,
		Status:   model.StatusBlocked,
		Commit:   "85f8c029b902ed9400bc600bac301a0aadb144ac",
	})
	s.CreateBuild(&model.Build{
		RepoID:   repo.ID,
		ConfigID: conf.ID,
		Status:   model.StatusPending,
		Commit:   "85f8c029b902ed9400bc600bac301a0aadb144ac",
	})

	if ok, _ := s.ConfigFindApproved(conf); ok == true {
		t.Errorf("Want config not approved, when blocked or pending")
		return
	}

	s.CreateBuild(&model.Build{
		RepoID:   repo.ID,
		ConfigID: conf.ID,
		Status:   model.StatusRunning,
		Commit:   "85f8c029b902ed9400bc600bac301a0aadb144ac",
	})

	if ok, _ := s.ConfigFindApproved(conf); ok == false {
		t.Errorf("Want config approved, when running.")
		return
	}
}

func TestConfigIndexes(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from config")
		s.Close()
	}()

	var (
		data = "pipeline: [ { image: golang, commands: [ go build, go test ] } ]"
		hash = "8d8647c9aa90d893bfb79dddbe901f03e258588121e5202632f8ae5738590b26"
	)

	if err := s.ConfigCreate(
		&model.Config{
			RepoID: 2,
			Data:   data,
			Hash:   hash,
		},
	); err != nil {
		t.Errorf("Unexpected error: insert config: %s", err)
		return
	}

	// fail due to duplicate sha
	if err := s.ConfigCreate(
		&model.Config{
			RepoID: 2,
			Data:   data,
			Hash:   hash,
		},
	); err == nil {
		t.Errorf("Unexpected error: dupliate sha")
	}
}
