package bolt

import (
	"os"
	"testing"

	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
)

func TestUser(t *testing.T) {
	g := Goblin(t)
	g.Describe("Users", func() {
		var db *DB // temporary database

		// create a new database before each unit
		// test and destroy afterwards.
		g.BeforeEach(func() {
			db = Must("/tmp/drone.test.db")
		})
		g.AfterEach(func() {
			os.Remove(db.Path())
		})

		g.It("Should find", func() {
			db.InsertUser(&common.User{Login: "octocat"})
			user, err := db.GetUser("octocat")
			g.Assert(err).Equal(nil)
			g.Assert(user.Login).Equal("octocat")
		})

		g.It("Should insert", func() {
			err := db.InsertUser(&common.User{Login: "octocat"})
			g.Assert(err).Equal(nil)

			user, err := db.GetUser("octocat")
			g.Assert(err).Equal(nil)
			g.Assert(user.Login).Equal("octocat")
			g.Assert(user.Created != 0).IsTrue()
			g.Assert(user.Updated != 0).IsTrue()
		})

		g.It("Should not insert if exists", func() {
			db.InsertUser(&common.User{Login: "octocat"})
			err := db.InsertUser(&common.User{Login: "octocat"})
			g.Assert(err).Equal(ErrKeyExists)
		})

		g.It("Should update", func() {
			db.InsertUser(&common.User{Login: "octocat"})
			user, err := db.GetUser("octocat")
			g.Assert(err).Equal(nil)

			user.Email = "octocat@github.com"
			err = db.UpdateUser(user)
			g.Assert(err).Equal(nil)

			user_, err := db.GetUser("octocat")
			g.Assert(err).Equal(nil)
			g.Assert(user_.Login).Equal(user.Login)
			g.Assert(user_.Email).Equal(user.Email)
		})

		g.It("Should delete", func() {
			db.InsertUser(&common.User{Login: "octocat"})
			user, err := db.GetUser("octocat")
			g.Assert(err).Equal(nil)

			err = db.DeleteUser(user)
			g.Assert(err).Equal(nil)

			_, err = db.GetUser("octocat")
			g.Assert(err).Equal(ErrKeyNotFound)
		})

		g.It("Should list", func() {
			db.InsertUser(&common.User{Login: "bert"})
			db.InsertUser(&common.User{Login: "ernie"})
			users, err := db.GetUserList()
			g.Assert(err).Equal(nil)
			g.Assert(len(users)).Equal(2)
		})

		g.It("Should count", func() {
			db.InsertUser(&common.User{Login: "bert"})
			db.InsertUser(&common.User{Login: "ernie"})
			count, err := db.GetUserCount()
			g.Assert(err).Equal(nil)
			g.Assert(count).Equal(2)
		})
	})
}
