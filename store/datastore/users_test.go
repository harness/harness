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

func TestUsers(t *testing.T) {
	s := newTest()
	defer s.Close()

	g := goblin.Goblin(t)
	g.Describe("User", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			s.Exec("DELETE FROM users")
			s.Exec("DELETE FROM repos")
			s.Exec("DELETE FROM builds")
			s.Exec("DELETE FROM procs")
		})

		g.It("Should Update a User", func() {
			user := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err1 := s.CreateUser(&user)
			err2 := s.UpdateUser(&user)
			getuser, err3 := s.GetUser(user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
		})

		g.It("Should Add a new User", func() {
			user := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err := s.CreateUser(&user)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID != 0).IsTrue()
		})

		g.It("Should Get a User", func() {
			user := model.User{
				Login:  "joe",
				Token:  "f0b461ca586c27872b43a0685cbc2847",
				Secret: "976f22a5eef7caacb7e678d6c52f49b1",
				Email:  "foo@bar.com",
				Avatar: "b9015b0857e16ac4d94a0ffd9a0b79c8",
				Active: true,
			}

			s.CreateUser(&user)
			getuser, err := s.GetUser(user.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
			g.Assert(user.Token).Equal(getuser.Token)
			g.Assert(user.Secret).Equal(getuser.Secret)
			g.Assert(user.Email).Equal(getuser.Email)
			g.Assert(user.Avatar).Equal(getuser.Avatar)
			g.Assert(user.Active).Equal(getuser.Active)
		})

		g.It("Should Get a User By Login", func() {
			user := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			s.CreateUser(&user)
			getuser, err := s.GetUserLogin(user.Login)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
		})

		g.It("Should Enforce Unique User Login", func() {
			user1 := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			user2 := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			err1 := s.CreateUser(&user1)
			err2 := s.CreateUser(&user2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should Get a User List", func() {
			user1 := model.User{
				Login: "jane",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			s.CreateUser(&user1)
			s.CreateUser(&user2)
			users, err := s.GetUserList()
			g.Assert(err == nil).IsTrue()
			g.Assert(len(users)).Equal(2)
			g.Assert(users[0].Login).Equal(user1.Login)
			g.Assert(users[0].Email).Equal(user1.Email)
			g.Assert(users[0].Token).Equal(user1.Token)
		})

		g.It("Should Get a User Count", func() {
			user1 := model.User{
				Login: "jane",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			s.CreateUser(&user1)
			s.CreateUser(&user2)
			count, err := s.GetUserCount()
			g.Assert(err == nil).IsTrue()
			if s.driver != "postgres" {
				// we have to skip this check for postgres because it uses
				// an estimate which may not be updated.
				g.Assert(count).Equal(2)
			}
		})

		g.It("Should Get a User Count Zero", func() {
			count, err := s.GetUserCount()
			g.Assert(err == nil).IsTrue()
			g.Assert(count).Equal(0)
		})

		g.It("Should Del a User", func() {
			user := model.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			s.CreateUser(&user)
			_, err1 := s.GetUser(user.ID)
			err2 := s.DeleteUser(&user)
			_, err3 := s.GetUser(user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should get the Build feed for a User", func() {
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
				Status: model.StatusSuccess,
			}
			build3 := &model.Build{
				RepoID: repo2.ID,
				Status: model.StatusSuccess,
			}
			build4 := &model.Build{
				RepoID: repo3.ID,
				Status: model.StatusSuccess,
			}
			s.CreateBuild(build1)
			s.CreateBuild(build2)
			s.CreateBuild(build3)
			s.CreateBuild(build4)

			builds, err := s.UserFeed(user)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(builds)).Equal(3)
			g.Assert(builds[0].FullName).Equal(repo2.FullName)
			g.Assert(builds[1].FullName).Equal(repo1.FullName)
			g.Assert(builds[2].FullName).Equal(repo1.FullName)
		})
	})
}
