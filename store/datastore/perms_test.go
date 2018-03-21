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

func TestPermFind(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from perms")
		s.Exec("delete from repos")
		s.Close()
	}()

	user := &model.User{ID: 1}
	repo := &model.Repo{
		UserID:   1,
		FullName: "bradrydzewski/drone",
		Owner:    "bradrydzewski",
		Name:     "drone",
	}
	s.CreateRepo(repo)

	err := s.PermUpsert(
		&model.Perm{
			UserID: user.ID,
			RepoID: repo.ID,
			Repo:   repo.FullName,
			Pull:   true,
			Push:   false,
			Admin:  false,
		},
	)
	if err != nil {
		t.Error(err)
		return
	}

	perm, err := s.PermFind(user, repo)
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := perm.Pull, true; got != want {
		t.Errorf("Wanted pull %v, got %v", want, got)
	}
	if got, want := perm.Push, false; got != want {
		t.Errorf("Wanted push %v, got %v", want, got)
	}
	if got, want := perm.Admin, false; got != want {
		t.Errorf("Wanted admin %v, got %v", want, got)
	}
}

func TestPermUpsert(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from perms")
		s.Exec("delete from repos")
		s.Close()
	}()

	user := &model.User{ID: 1}
	repo := &model.Repo{
		UserID:   1,
		FullName: "bradrydzewski/drone",
		Owner:    "bradrydzewski",
		Name:     "drone",
	}
	s.CreateRepo(repo)

	err := s.PermUpsert(
		&model.Perm{
			UserID: user.ID,
			RepoID: repo.ID,
			Repo:   repo.FullName,
			Pull:   true,
			Push:   false,
			Admin:  false,
		},
	)
	if err != nil {
		t.Error(err)
		return
	}

	perm, err := s.PermFind(user, repo)
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := perm.Pull, true; got != want {
		t.Errorf("Wanted pull %v, got %v", want, got)
	}
	if got, want := perm.Push, false; got != want {
		t.Errorf("Wanted push %v, got %v", want, got)
	}
	if got, want := perm.Admin, false; got != want {
		t.Errorf("Wanted admin %v, got %v", want, got)
	}

	//
	// this will attempt to replace the existing permissions
	// using the insert or replace logic.
	//

	err = s.PermUpsert(
		&model.Perm{
			UserID: user.ID,
			RepoID: repo.ID,
			Repo:   repo.FullName,
			Pull:   true,
			Push:   true,
			Admin:  true,
		},
	)
	if err != nil {
		t.Error(err)
		return
	}

	perm, err = s.PermFind(user, repo)
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := perm.Pull, true; got != want {
		t.Errorf("Wanted pull %v, got %v", want, got)
	}
	if got, want := perm.Push, true; got != want {
		t.Errorf("Wanted push %v, got %v", want, got)
	}
	if got, want := perm.Admin, true; got != want {
		t.Errorf("Wanted admin %v, got %v", want, got)
	}
}

func TestPermDelete(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from perms")
		s.Exec("delete from repos")
		s.Close()
	}()

	user := &model.User{ID: 1}
	repo := &model.Repo{
		UserID:   1,
		FullName: "bradrydzewski/drone",
		Owner:    "bradrydzewski",
		Name:     "drone",
	}
	s.CreateRepo(repo)

	err := s.PermUpsert(
		&model.Perm{
			UserID: user.ID,
			RepoID: repo.ID,
			Repo:   repo.FullName,
			Pull:   true,
			Push:   false,
			Admin:  false,
		},
	)
	if err != nil {
		t.Errorf("Unexpected error: insert perm: %s", err)
		return
	}

	perm, err := s.PermFind(user, repo)
	if err != nil {
		t.Errorf("Unexpected error: select perm: %s", err)
		return
	}
	err = s.PermDelete(perm)
	if err != nil {
		t.Errorf("Unexpected error: delete perm: %s", err)
		return
	}
	_, err = s.PermFind(user, repo)
	if err == nil {
		t.Errorf("Expect error: sql.ErrNoRows")
		return
	}
}
