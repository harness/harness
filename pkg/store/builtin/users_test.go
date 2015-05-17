package builtin

import (
	"testing"

	common "github.com/drone/drone/pkg/types"
	"github.com/franela/goblin"
)

func TestUserstore(t *testing.T) {
	db := mustConnectTest()
	us := NewUserstore(db)
	cs := NewCommitstore(db)
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
			db.Exec("DELETE FROM tasks")
		})

		g.It("Should Update a User", func() {
			user := common.User{
				Login: "joe",
				Name:  "Joe Sixpack",
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
			user := common.User{
				Login: "joe",
				Name:  "Joe Sixpack",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			err := us.AddUser(&user)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID != 0).IsTrue()
		})

		g.It("Should Get a User", func() {
			user := common.User{
				Login:    "joe",
				Token:    "f0b461ca586c27872b43a0685cbc2847",
				Secret:   "976f22a5eef7caacb7e678d6c52f49b1",
				Name:     "Joe Sixpack",
				Email:    "foo@bar.com",
				Gravatar: "b9015b0857e16ac4d94a0ffd9a0b79c8",
				Active:   true,
				Admin:    true,
				Created:  1398065343,
				Updated:  1398065344,
			}
			us.AddUser(&user)
			getuser, err := us.User(user.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
			g.Assert(user.Token).Equal(getuser.Token)
			g.Assert(user.Secret).Equal(getuser.Secret)
			g.Assert(user.Name).Equal(getuser.Name)
			g.Assert(user.Email).Equal(getuser.Email)
			g.Assert(user.Gravatar).Equal(getuser.Gravatar)
			g.Assert(user.Active).Equal(getuser.Active)
			g.Assert(user.Admin).Equal(getuser.Admin)
			g.Assert(user.Created).Equal(getuser.Created)
			g.Assert(user.Updated).Equal(getuser.Updated)
		})

		g.It("Should Get a User By Login", func() {
			user := common.User{
				Login: "joe",
				Name:  "Joe Sixpack",
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
			user1 := common.User{
				Login: "joe",
				Name:  "Joe Sixpack",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			user2 := common.User{
				Login: "joe",
				Name:  "Joe Sixpack",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			err1 := us.AddUser(&user1)
			err2 := us.AddUser(&user2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should Get a User List", func() {
			user1 := common.User{
				Login: "jane",
				Name:  "Jane Doe",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := common.User{
				Login: "joe",
				Name:  "Joe Sixpack",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			us.AddUser(&user1)
			us.AddUser(&user2)
			users, err := us.UserList()
			g.Assert(err == nil).IsTrue()
			g.Assert(len(users)).Equal(2)
			g.Assert(users[0].Login).Equal(user1.Login)
			g.Assert(users[0].Name).Equal(user1.Name)
			g.Assert(users[0].Email).Equal(user1.Email)
			g.Assert(users[0].Token).Equal(user1.Token)
		})

		g.It("Should Get a User Count", func() {
			user1 := common.User{
				Login: "jane",
				Name:  "Jane Doe",
				Email: "foo@bar.com",
				Token: "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := common.User{
				Login: "joe",
				Name:  "Joe Sixpack",
				Email: "foo@bar.com",
				Token: "e42080dddf012c718e476da161d21ad5",
			}
			us.AddUser(&user1)
			us.AddUser(&user2)
			count, err := us.UserCount()
			g.Assert(err == nil).IsTrue()
			g.Assert(count).Equal(2)
		})

		g.It("Should Del a User", func() {
			user := common.User{
				Login: "joe",
				Name:  "Joe Sixpack",
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
			repo1 := &common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			repo2 := &common.Repo{
				UserID: 2,
				Owner:  "drone",
				Name:   "drone",
			}
			rs.AddRepo(repo1)
			rs.AddRepo(repo2)
			ss.AddStar(&common.User{ID: 1}, repo1)
			commit1 := &common.Commit{
				RepoID: 1,
				State:  common.StateFailure,
				Ref:    "refs/heads/master",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			commit2 := &common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/dev",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			commit3 := &common.Commit{
				RepoID: 2,
				State:  common.StateSuccess,
				Ref:    "refs/heads/dev",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			cs.AddCommit(commit1)
			cs.AddCommit(commit2)
			cs.AddCommit(commit3)
			commits, err := us.UserFeed(&common.User{ID: 1}, 20, 0)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(commits)).Equal(2)
			g.Assert(commits[0].State).Equal(commit2.State)
			g.Assert(commits[0].Owner).Equal("bradrydzewski")
			g.Assert(commits[0].Name).Equal("drone")
		})
	})
}
