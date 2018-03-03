// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Drone Enterpise License
// that can be found in the LICENSE file.

package secrets

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestExtends(t *testing.T) {
	base := &mocker{}
	base.list = []*model.Secret{
		{Name: "foo"},
		{Name: "bar"},
	}

	with := &mocker{}
	with.list = []*model.Secret{
		{Name: "baz"},
		{Name: "qux"},
	}

	extended := Extend(base, with)
	list, err := extended.SecretListBuild(nil, nil)
	if err != nil {
		t.Errorf("Expected combined secret list, got error %q", err)
	}

	if got, want := list[0], with.list[0]; got != want {
		t.Errorf("Expected correct precedence. Want %s, got %s", want.Name, got.Name)
	}
	if got, want := list[1], with.list[1]; got != want {
		t.Errorf("Expected correct precedence. Want %s, got %s", want.Name, got.Name)
	}
	if got, want := list[2], base.list[0]; got != want {
		t.Errorf("Expected correct precedence. Want %s, got %s", want.Name, got.Name)
	}
	if got, want := list[3], base.list[1]; got != want {
		t.Errorf("Expected correct precedence. Want %s, got %s", want.Name, got.Name)
	}
}

type mocker struct {
	list  []*model.Secret
	error error
}

func (m *mocker) SecretFind(*model.Repo, string) (*model.Secret, error) {
	return nil, nil
}
func (m *mocker) SecretList(*model.Repo) ([]*model.Secret, error) {
	return nil, nil
}
func (m *mocker) SecretListBuild(*model.Repo, *model.Build) ([]*model.Secret, error) {
	return m.list, m.error
}
func (m *mocker) SecretCreate(*model.Repo, *model.Secret) error {
	return nil
}
func (m *mocker) SecretUpdate(*model.Repo, *model.Secret) error {
	return nil
}
func (m *mocker) SecretDelete(*model.Repo, string) error {
	return nil
}
