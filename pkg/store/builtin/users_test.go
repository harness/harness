package builtin

import (
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	"github.com/drone/drone/pkg/types"
)

func TestUserstore(t *testing.T) {
	db := mustConnectTest()
	us := NewUserstore(db)
	cs := NewBuildstore(db)
	rs := NewRepostore(db)
	ss := NewStarstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Userstore", func() {

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
			user := types.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err1 := us.AddUser(&user)
			err2 := us.SetUser(&user)
			getuser, err3 := us.User(user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
		})

		g.It("Should Add a new User", func() {
			user := types.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err := us.AddUser(&user)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID != 0).IsTrue()
		})

		g.It("Should Get a User", func() {
			user := types.User{
				Login:  "joe",
				Token:  "f0b461ca586c27872b43a0685cbc2847",
				Secret: "976f22a5eef7caacb7e678d6c52f49b1",
				Email:  "foo@bar.com",
				Avatar: "b9015b0857e16ac4d94a0ffd9a0b79c8",
				Active: true,
				Admin:  true,
			}
			us.AddUser(&user)
			getuser, err := us.User(user.ID)
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
			user := types.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			us.AddUser(&user)
			getuser, err := us.UserLogin(user.Login)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
		})

		g.It("Should Enforce Unique User Login", func() {
			user1 := types.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			user2 := types.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			err1 := us.AddUser(&user1)
			err2 := us.AddUser(&user2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should Get a User List", func() {
			user1 := types.User{
				Login: "jane",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := types.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			us.AddUser(&user1)
			us.AddUser(&user2)
			users, err := us.UserList()
			g.Assert(err == nil).IsTrue()
			g.Assert(len(users)).Equal(2)
			g.Assert(users[0].Login).Equal(user1.Login)
			g.Assert(users[0].Email).Equal(user1.Email)
			g.Assert(users[0].Token).Equal(user1.Token)
		})

		g.It("Should Get a User Count", func() {
			user1 := types.User{
				Login: "jane",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := types.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			us.AddUser(&user1)
			us.AddUser(&user2)
			count, err := us.UserCount()
			g.Assert(err == nil).IsTrue()
			g.Assert(count).Equal(2)
		})

		g.It("Should Get a User Count Zero", func() {
			count, err := us.UserCount()
			g.Assert(err == nil).IsTrue()
			g.Assert(count).Equal(0)
		})

		g.It("Should Del a User", func() {
			user := types.User{
				Login: "joe",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			us.AddUser(&user)
			_, err1 := us.User(user.ID)
			err2 := us.DelUser(&user)
			_, err3 := us.User(user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should get the Build feed for a User", func() {
			repo1 := &types.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			repo2 := &types.Repo{
				UserID: 2,
				Owner:  "drone",
				Name:   "drone",
			}
			rs.AddRepo(repo1)
			rs.AddRepo(repo2)
			ss.AddStar(&types.User{ID: 1}, repo1)
			build1 := &types.Build{
				RepoID: 1,
				Status: types.StateFailure,
			}
			build2 := &types.Build{
				RepoID: 1,
				Status: types.StateSuccess,
			}
			build3 := &types.Build{
				RepoID: 2,
				Status: types.StateSuccess,
			}
			cs.AddBuild(build1)
			cs.AddBuild(build2)
			cs.AddBuild(build3)
			builds, err := us.UserFeed(&types.User{ID: 1}, 20, 0)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(builds)).Equal(2)
			//g.Assert(builds[0].Status).Equal(commit2.Status)
			g.Assert(builds[0].Owner).Equal("bradrydzewski")
			g.Assert(builds[0].Name).Equal("drone")
		})
	})
}
