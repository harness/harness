// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		t.Errorf("Want %d secrets, got %d", want, got)
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

func TestSecretDelete(t *testing.T) {
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

	if err := s.SecretDelete(secret); err != nil {
		t.Errorf("Unexpected error: delete secret: %s", err)
		return
	}
	_, err := s.SecretFind(&model.Repo{ID: 1}, "foo")
	if err == nil {
		t.Errorf("Expect error: sql.ErrNoRows")
		return
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
