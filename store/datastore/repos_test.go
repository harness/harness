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
	"github.com/franela/goblin"
)

func TestRepos(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("Repo", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM builds")
			db.Exec("DELETE FROM repos")
			db.Exec("DELETE FROM users")
		})

		g.It("Should Set a Repo", func() {
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err1 := s.CreateRepo(&repo)
			err2 := s.UpdateRepo(&repo)
			getrepo, err3 := s.GetRepo(repo.ID)

			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
		})

		g.It("Should Add a Repo", func() {
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err := s.CreateRepo(&repo)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID != 0).IsTrue()
		})

		g.It("Should Get a Repo by ID", func() {
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			s.CreateRepo(&repo)
			getrepo, err := s.GetRepo(repo.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Get a Repo by Name", func() {
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			s.CreateRepo(&repo)
			getrepo, err := s.GetRepoName(repo.FullName)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Enforce Unique Repo Name", func() {
			repo1 := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			repo2 := model.Repo{
				UserID:   2,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err1 := s.CreateRepo(&repo1)
			err2 := s.CreateRepo(&repo2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})
	})
}

func TestRepoList(t *testing.T) {
	s := newTest()
	s.Exec("delete from repos")
	s.Exec("delete from users")
	s.Exec("delete from perms")

	defer func() {
		s.Exec("delete from repos")
		s.Exec("delete from users")
		s.Exec("delete from perms")
		s.Close()
	}()

	user := &model.User{
		Login: "joe",
		Email: "foo@bar.com",
		Token: "e42080dddf012c718e476da161d21ad5",
	}
	s.CreateUser(user)

	repo1 := &model.Repo{
		Owner:    "bradrydzewski",
		Name:     "drone",
		FullName: "bradrydzewski/drone",
	}
	repo2 := &model.Repo{
		Owner:    "drone",
		Name:     "drone",
		FullName: "drone/drone",
	}
	repo3 := &model.Repo{
		Owner:    "octocat",
		Name:     "hello-world",
		FullName: "octocat/hello-world",
	}
	s.CreateRepo(repo1)
	s.CreateRepo(repo2)
	s.CreateRepo(repo3)

	s.PermBatch([]*model.Perm{
		{UserID: user.ID, Repo: repo1.FullName},
		{UserID: user.ID, Repo: repo2.FullName},
	})

	repos, err := s.RepoList(user)
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := len(repos), 2; got != want {
		t.Errorf("Want %d repositories, got %d", want, got)
	}
	if got, want := repos[0].ID, repo1.ID; got != want {
		t.Errorf("Want repository id %d, got %d", want, got)
	}
	if got, want := repos[1].ID, repo2.ID; got != want {
		t.Errorf("Want repository id %d, got %d", want, got)
	}
}

func TestRepoListLatest(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from repos")
		s.Exec("delete from users")
		s.Exec("delete from perms")
		s.Close()
	}()

	user := &model.User{
		Login: "joe",
		Email: "foo@bar.com",
		Token: "e42080dddf012c718e476da161d21ad5",
	}
	s.CreateUser(user)

	repo1 := &model.Repo{
		Owner:    "bradrydzewski",
		Name:     "drone",
		FullName: "bradrydzewski/drone",
		IsActive: true,
	}
	repo2 := &model.Repo{
		Owner:    "drone",
		Name:     "drone",
		FullName: "drone/drone",
		IsActive: true,
	}
	repo3 := &model.Repo{
		Owner:    "octocat",
		Name:     "hello-world",
		FullName: "octocat/hello-world",
		IsActive: true,
	}
	s.CreateRepo(repo1)
	s.CreateRepo(repo2)
	s.CreateRepo(repo3)

	s.PermBatch([]*model.Perm{
		{UserID: user.ID, Repo: repo1.FullName},
		{UserID: user.ID, Repo: repo2.FullName},
	})

	build1 := &model.Build{
		RepoID: repo1.ID,
		Status: model.StatusFailure,
	}
	build2 := &model.Build{
		RepoID: repo1.ID,
		Status: model.StatusRunning,
	}
	build3 := &model.Build{
		RepoID: repo2.ID,
		Status: model.StatusKilled,
	}
	build4 := &model.Build{
		RepoID: repo3.ID,
		Status: model.StatusError,
	}
	s.CreateBuild(build1)
	s.CreateBuild(build2)
	s.CreateBuild(build3)
	s.CreateBuild(build4)

	builds, err := s.RepoListLatest(user)
	if err != nil {
		t.Errorf("Unexpected error: repository list with latest build: %s", err)
		return
	}
	if got, want := len(builds), 2; got != want {
		t.Errorf("Want %d repositories, got %d", want, got)
	}
	if got, want := builds[0].Status, model.StatusRunning; want != got {
		t.Errorf("Want repository status %s, got %s", want, got)
	}
	if got, want := builds[0].FullName, repo1.FullName; want != got {
		t.Errorf("Want repository name %s, got %s", want, got)
	}
	if got, want := builds[1].Status, model.StatusKilled; want != got {
		t.Errorf("Want repository status %s, got %s", want, got)
	}
	if got, want := builds[1].FullName, repo2.FullName; want != got {
		t.Errorf("Want repository name %s, got %s", want, got)
	}
}

func TestRepoCount(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from repos")
		s.Exec("delete from users")
		s.Exec("delete from perms")
		s.Close()
	}()

	repo1 := &model.Repo{
		Owner:    "bradrydzewski",
		Name:     "drone",
		FullName: "bradrydzewski/drone",
		IsActive: true,
	}
	repo2 := &model.Repo{
		Owner:    "drone",
		Name:     "drone",
		FullName: "drone/drone",
		IsActive: true,
	}
	repo3 := &model.Repo{
		Owner:    "drone",
		Name:     "drone-ui",
		FullName: "drone/drone-ui",
		IsActive: false,
	}
	s.CreateRepo(repo1)
	s.CreateRepo(repo2)
	s.CreateRepo(repo3)

	s.Exec("ANALYZE")
	count, _ := s.GetRepoCount()
	if got, want := count, 2; got != want {
		t.Errorf("Want %d repositories, got %d", want, got)
	}
}

func TestRepoBatch(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from repos")
		s.Exec("delete from users")
		s.Exec("delete from perms")
		s.Close()
	}()

	repo := &model.Repo{
		UserID:   1,
		FullName: "foo/bar",
		Owner:    "foo",
		Name:     "bar",
		IsActive: true,
	}
	err := s.CreateRepo(repo)
	if err != nil {
		t.Error(err)
		return
	}

	err = s.RepoBatch(
		[]*model.Repo{
			{
				UserID:   1,
				FullName: "foo/bar",
				Owner:    "foo",
				Name:     "bar",
				IsActive: true,
			},
			{
				UserID:   1,
				FullName: "bar/baz",
				Owner:    "bar",
				Name:     "baz",
				IsActive: true,
			},
			{
				UserID:   1,
				FullName: "baz/qux",
				Owner:    "baz",
				Name:     "qux",
				IsActive: true,
			},
		},
	)
	if err != nil {
		t.Error(err)
		return
	}

	s.Exec("ANALYZE")
	count, _ := s.GetRepoCount()
	if got, want := count, 3; got != want {
		t.Errorf("Want %d repositories, got %d", want, got)
	}
}

func TestRepoCrud(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from repos")
		s.Exec("delete from users")
		s.Exec("delete from perms")
		s.Close()
	}()

	repo := model.Repo{
		UserID:   1,
		FullName: "bradrydzewski/drone",
		Owner:    "bradrydzewski",
		Name:     "drone",
	}
	s.CreateRepo(&repo)
	_, err1 := s.GetRepo(repo.ID)
	err2 := s.DeleteRepo(&repo)
	_, err3 := s.GetRepo(repo.ID)
	if err1 != nil {
		t.Errorf("Unexpected error: select repository: %s", err1)
	}
	if err2 != nil {
		t.Errorf("Unexpected error: delete repository: %s", err2)
	}
	if err3 == nil {
		t.Errorf("Expected error: sql.ErrNoRows")
	}
}
