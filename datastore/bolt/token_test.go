package bolt

import (
	"os"
	"testing"

	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
)

func TestToken(t *testing.T) {
	g := Goblin(t)
	g.Describe("Tokens", func() {
		var db *DB // temporary database

		// create a new database before each unit
		// test and destroy afterwards.
		g.BeforeEach(func() {
			db = Must("/tmp/drone.test.db")
		})
		g.AfterEach(func() {
			os.Remove(db.Path())
		})

		g.It("Should list for user", func() {
			db.SetUserNotExists(&common.User{Login: "octocat"})
			err1 := db.SetToken(&common.Token{Login: "octocat", Label: "gist"})
			err2 := db.SetToken(&common.Token{Login: "octocat", Label: "github"})
			g.Assert(err1).Equal(nil)
			g.Assert(err2).Equal(nil)

			list, err := db.TokenList("octocat")
			g.Assert(err).Equal(nil)
			g.Assert(len(list)).Equal(2)
		})

		g.It("Should insert", func() {
			db.SetUserNotExists(&common.User{Login: "octocat"})
			err := db.SetToken(&common.Token{Login: "octocat", Label: "gist"})
			g.Assert(err).Equal(nil)

			token, err := db.Token("octocat", "gist")
			g.Assert(err).Equal(nil)
			g.Assert(token.Label).Equal("gist")
			g.Assert(token.Login).Equal("octocat")
		})

		g.It("Should delete", func() {
			db.SetUserNotExists(&common.User{Login: "octocat"})
			err := db.SetToken(&common.Token{Login: "octocat", Label: "gist"})
			g.Assert(err).Equal(nil)

			token, err := db.Token("octocat", "gist")
			g.Assert(err).Equal(nil)
			g.Assert(token.Label).Equal("gist")
			g.Assert(token.Login).Equal("octocat")

			err = db.DelToken(token)
			g.Assert(err).Equal(nil)

			token, err = db.Token("octocat", "gist")
			g.Assert(err != nil).IsTrue()
		})
	})
}
