package datastore

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestSecretFind(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from secrets")
		s.Close()
	}()

	err := s.SecretCreate(&model.Secret{
		RepoID: 1,
		Name:   "password",
		Value:  "correct-horse-battery-staple",
		Images: []string{"golang", "node"},
		Events: []string{"push", "tag"},
	})
	if err != nil {
		t.Errorf("Unexpected error: insert secret: %s", err)
		return
	}

	secret, err := s.SecretFind(&model.Repo{ID: 1}, "password")
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := secret.RepoID, int64(1); got != want {
		t.Errorf("Want repo id %d, got %d", want, got)
	}
	if got, want := secret.Name, "password"; got != want {
		t.Errorf("Want secret name %s, got %s", want, got)
	}
	if got, want := secret.Value, "correct-horse-battery-staple"; got != want {
		t.Errorf("Want secret value %s, got %s", want, got)
	}
	if got, want := secret.Events[0], "push"; got != want {
		t.Errorf("Want secret event %s, got %s", want, got)
	}
	if got, want := secret.Events[1], "tag"; got != want {
		t.Errorf("Want secret event %s, got %s", want, got)
	}
	if got, want := secret.Images[0], "golang"; got != want {
		t.Errorf("Want secret image %s, got %s", want, got)
	}
	if got, want := secret.Images[1], "node"; got != want {
		t.Errorf("Want secret image %s, got %s", want, got)
	}
}

func TestSecretList(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from secrets")
		s.Close()
	}()

	s.SecretCreate(&model.Secret{
		RepoID: 1,
		Name:   "foo",
		Value:  "bar",
	})
	s.SecretCreate(&model.Secret{
		RepoID: 1,
		Name:   "baz",
		Value:  "qux",
	})

	list, err := s.SecretList(&model.Repo{ID: 1})
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := len(list), 2; got != want {
		t.Errorf("Want %d registries, got %d", want, got)
	}
}

func TestSecretUpdate(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from secrets")
		s.Close()
	}()

	secret := &model.Secret{
		RepoID: 1,
		Name:   "foo",
		Value:  "baz",
	}
	if err := s.SecretCreate(secret); err != nil {
		t.Errorf("Unexpected error: insert secret: %s", err)
		return
	}
	secret.Value = "qux"
	if err := s.SecretUpdate(secret); err != nil {
		t.Errorf("Unexpected error: update secret: %s", err)
		return
	}
	updated, err := s.SecretFind(&model.Repo{ID: 1}, "foo")
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := updated.Value, "qux"; got != want {
		t.Errorf("Want secret value %s, got %s", want, got)
	}
}

func TestSecretIndexes(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from secrets")
		s.Close()
	}()

	if err := s.SecretCreate(&model.Secret{
		RepoID: 1,
		Name:   "foo",
		Value:  "bar",
	}); err != nil {
		t.Errorf("Unexpected error: insert secret: %s", err)
		return
	}

	// fail due to duplicate name
	if err := s.SecretCreate(&model.Secret{
		RepoID: 1,
		Name:   "foo",
		Value:  "baz",
	}); err == nil {
		t.Errorf("Unexpected error: dupliate name")
	}
}
