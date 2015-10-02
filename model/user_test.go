package model

import (
	"testing"

	"github.com/drone/drone/shared/database"
	"github.com/franela/goblin"
)

func TestUserstore(t *testing.T) {
	db := database.Open("sqlite3", ":memory:")
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("User", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM users")
			db.Exec("DELETE FROM stars")
			db.Exec("DELETE FROM repos")
			db.Exec("DELETE FROM builds")
			db.Exec("DELETE FROM jobs")
		})

		g.It("Should Update a User", func() {
			user := User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err1 := CreateUser(db, &user)
			err2 := UpdateUser(db, &user)
			getuser, err3 := GetUser(db, user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
		})

		g.It("Should Add a new User", func() {
			user := User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err := CreateUser(db, &user)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID != 0).IsTrue()
		})

		g.It("Should Get a User", func() {
			user := User{
				Login:  "joe",
				Token:  "f0b461ca586c27872b43a0685cbc2847",
				Secret: "976f22a5eef7caacb7e678d6c52f49b1",
				Email:  "foo@bar.com",
				Avatar: "b9015b0857e16ac4d94a0ffd9a0b79c8",
				Active: true,
				Admin:  true,
			}

			CreateUser(db, &user)
			getuser, err := GetUser(db, user.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
			g.Assert(user.Token).Equal(getuser.Token)
			g.Assert(user.Secret).Equal(getuser.Secret)
			g.Assert(user.Email).Equal(getuser.Email)
			g.Assert(user.Avatar).Equal(getuser.Avatar)
			g.Assert(user.Active).Equal(getuser.Active)
			g.Assert(user.Admin).Equal(getuser.Admin)
		})

		g.It("Should Get a User By Login", func() {
			user := User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			CreateUser(db, &user)
			getuser, err := GetUserLogin(db, user.Login)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
		})

		g.It("Should Enforce Unique User Login", func() {
			user1 := User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			user2 := User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			err1 := CreateUser(db, &user1)
			err2 := CreateUser(db, &user2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should Get a User List", func() {
			user1 := User{
				Login: "jane",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			CreateUser(db, &user1)
			CreateUser(db, &user2)
			users, err := GetUserList(db)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(users)).Equal(2)
			g.Assert(users[0].Login).Equal(user1.Login)
			g.Assert(users[0].Email).Equal(user1.Email)
			g.Assert(users[0].Token).Equal(user1.Token)
		})

		g.It("Should Get a User Count", func() {
			user1 := User{
				Login: "jane",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			CreateUser(db, &user1)
			CreateUser(db, &user2)
			count, err := GetUserCount(db)
			g.Assert(err == nil).IsTrue()
			g.Assert(count).Equal(2)
		})

		g.It("Should Get a User Count Zero", func() {
			count, err := GetUserCount(db)
			g.Assert(err == nil).IsTrue()
			g.Assert(count).Equal(0)
		})

		g.It("Should Del a User", func() {
			user := User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			CreateUser(db, &user)
			_, err1 := GetUser(db, user.ID)
			err2 := DeleteUser(db, &user)
			_, err3 := GetUser(db, user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should get the Build feed for a User", func() {
			repo1 := &Repo{
				UserID:   1,
				Owner:    "bradrydzewski",
				Name:     "drone",
				FullName: "bradrydzewski/drone",
			}
			repo2 := &Repo{
				UserID:   2,
				Owner:    "drone",
				Name:     "drone",
				FullName: "drone/drone",
			}
			CreateRepo(db, repo1)
			CreateRepo(db, repo2)

			build1 := &Build{
				RepoID: repo1.ID,
				Status: StatusFailure,
				Author: "bradrydzewski",
			}
			build2 := &Build{
				RepoID: repo1.ID,
				Status: StatusSuccess,
				Author: "bradrydzewski",
			}
			build3 := &Build{
				RepoID: repo2.ID,
				Status: StatusSuccess,
				Author: "octocat",
			}
			CreateBuild(db, build1)
			CreateBuild(db, build2)
			CreateBuild(db, build3)

			builds, err := GetUserFeed(db, &User{ID: 1, Login: "bradrydzewski"}, 20, 0)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(builds)).Equal(2)
			g.Assert(builds[0].Owner).Equal("bradrydzewski")
			g.Assert(builds[0].Name).Equal("drone")
		})
	})
}
