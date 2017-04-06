package datastore

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestRegistryFind(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from registry")
		s.Close()
	}()

	err := s.RegistryCreate(&model.Registry{
		RepoID:   1,
		Address:  "index.docker.io",
		Username: "foo",
		Password: "bar",
		Email:    "foo@bar.com",
		Token:    "12345",
	})
	if err != nil {
		t.Errorf("Unexpected error: insert registry: %s", err)
		return
	}

	registry, err := s.RegistryFind(&model.Repo{ID: 1}, "index.docker.io")
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := registry.RepoID, int64(1); got != want {
		t.Errorf("Want repo id %d, got %d", want, got)
	}
	if got, want := registry.Address, "index.docker.io"; got != want {
		t.Errorf("Want registry address %s, got %s", want, got)
	}
	if got, want := registry.Username, "foo"; got != want {
		t.Errorf("Want registry username %s, got %s", want, got)
	}
	if got, want := registry.Password, "bar"; got != want {
		t.Errorf("Want registry password %s, got %s", want, got)
	}
	if got, want := registry.Email, "foo@bar.com"; got != want {
		t.Errorf("Want registry email %s, got %s", want, got)
	}
	if got, want := registry.Token, "12345"; got != want {
		t.Errorf("Want registry token %s, got %s", want, got)
	}
}

func TestRegistryList(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from registry")
		s.Close()
	}()

	s.RegistryCreate(&model.Registry{
		RepoID:   1,
		Address:  "index.docker.io",
		Username: "foo",
		Password: "bar",
	})
	s.RegistryCreate(&model.Registry{
		RepoID:   1,
		Address:  "foo.docker.io",
		Username: "foo",
		Password: "bar",
	})

	list, err := s.RegistryList(&model.Repo{ID: 1})
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := len(list), 2; got != want {
		t.Errorf("Want %d registries, got %d", want, got)
	}
}

func TestRegistryUpdate(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from registry")
		s.Close()
	}()

	registry := &model.Registry{
		RepoID:   1,
		Address:  "index.docker.io",
		Username: "foo",
		Password: "bar",
	}
	if err := s.RegistryCreate(registry); err != nil {
		t.Errorf("Unexpected error: insert registry: %s", err)
		return
	}
	registry.Password = "qux"
	if err := s.RegistryUpdate(registry); err != nil {
		t.Errorf("Unexpected error: update registry: %s", err)
		return
	}
	updated, err := s.RegistryFind(&model.Repo{ID: 1}, "index.docker.io")
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := updated.Password, "qux"; got != want {
		t.Errorf("Want registry password %s, got %s", want, got)
	}
}

func TestRegistryIndexes(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from registry")
		s.Close()
	}()

	if err := s.RegistryCreate(&model.Registry{
		RepoID:   1,
		Address:  "index.docker.io",
		Username: "foo",
		Password: "bar",
	}); err != nil {
		t.Errorf("Unexpected error: insert registry: %s", err)
		return
	}

	// fail due to duplicate addr
	if err := s.RegistryCreate(&model.Registry{
		RepoID:   1,
		Address:  "index.docker.io",
		Username: "baz",
		Password: "qux",
	}); err == nil {
		t.Errorf("Unexpected error: dupliate address")
	}
}
