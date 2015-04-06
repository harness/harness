package database

import (
	"testing"

	"github.com/drone/drone/shared/model"
	"github.com/franela/goblin"
)

func TestUserstore(t *testing.T) {
	db := mustConnectTest()
	us := NewUserstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Userstore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM users")
		})

		g.It("Should Put a User", func() {
			user := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			err := us.PostUser(&user)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID != 0).IsTrue()
		})

		g.It("Should Post a User", func() {
			user := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			err := us.PostUser(&user)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID != 0).IsTrue()
		})

		g.It("Should Get a User", func() {
			user := model.User{
				Login:    "joe",
				Remote:   "github.com",
				Access:   "f0b461ca586c27872b43a0685cbc2847",
				Secret:   "976f22a5eef7caacb7e678d6c52f49b1",
				Name:     "Joe Sixpack",
				Email:    "foo@bar.com",
				Gravatar: "b9015b0857e16ac4d94a0ffd9a0b79c8",
				Token:    "e42080dddf012c718e476da161d21ad5",
				Active:   true,
				Admin:    true,
				Created:  1398065343,
				Updated:  1398065344,
				Synced:   1398065345,
			}
			us.PostUser(&user)
			getuser, err := us.GetUser(user.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
			g.Assert(user.Remote).Equal(getuser.Remote)
			g.Assert(user.Access).Equal(getuser.Access)
			g.Assert(user.Secret).Equal(getuser.Secret)
			g.Assert(user.Name).Equal(getuser.Name)
			g.Assert(user.Email).Equal(getuser.Email)
			g.Assert(user.Gravatar).Equal(getuser.Gravatar)
			g.Assert(user.Token).Equal(getuser.Token)
			g.Assert(user.Active).Equal(getuser.Active)
			g.Assert(user.Admin).Equal(getuser.Admin)
			g.Assert(user.Created).Equal(getuser.Created)
			g.Assert(user.Updated).Equal(getuser.Updated)
			g.Assert(user.Synced).Equal(getuser.Synced)
		})

		g.It("Should Get a User By Login", func() {
			user := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			us.PostUser(&user)
			getuser, err := us.GetUserLogin(user.Remote, user.Login)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Login).Equal(getuser.Login)
			g.Assert(user.Remote).Equal(getuser.Remote)
		})

		g.It("Should Get a User By Token", func() {
			user := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			us.PostUser(&user)
			getuser, err := us.GetUserToken(user.Token)
			g.Assert(err == nil).IsTrue()
			g.Assert(user.ID).Equal(getuser.ID)
			g.Assert(user.Token).Equal(getuser.Token)
		})

		g.It("Should Enforce Unique User Token", func() {
			user1 := model.User{
				Login:  "jane",
				Remote: "github.com",
				Name:   "Jane Doe",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			user2 := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			err1 := us.PostUser(&user1)
			err2 := us.PostUser(&user2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should Enforce Unique User Remote and Login", func() {
			user1 := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			user2 := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "ab20g0ddaf012c744e136da16aa21ad9",
			}
			err1 := us.PostUser(&user1)
			err2 := us.PostUser(&user2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should Get a User List", func() {
			user1 := model.User{
				Login:  "jane",
				Remote: "github.com",
				Name:   "Jane Doe",
				Email:  "foo@bar.com",
				Token:  "ab20g0ddaf012c744e136da16aa21ad9",
			}
			user2 := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			us.PostUser(&user1)
			us.PostUser(&user2)
			users, err := us.GetUserList()
			g.Assert(err == nil).IsTrue()
			g.Assert(len(users)).Equal(2)
			g.Assert(users[0].Login).Equal(user1.Login)
			g.Assert(users[0].Remote).Equal(user1.Remote)
			g.Assert(users[0].Name).Equal(user1.Name)
			g.Assert(users[0].Email).Equal(user1.Email)
			g.Assert(users[0].Token).Equal(user1.Token)
		})

		g.It("Should Del a User", func() {
			user := model.User{
				Login:  "joe",
				Remote: "github.com",
				Name:   "Joe Sixpack",
				Email:  "foo@bar.com",
				Token:  "e42080dddf012c718e476da161d21ad5",
			}
			us.PostUser(&user)
			_, err1 := us.GetUser(user.ID)
			err2 := us.DelUser(&user)
			_, err3 := us.GetUser(user.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

	})
}
