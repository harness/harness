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

		g.It("Should find by label")
		g.It("Should list for user", func() {
			db.InsertUser(&common.User{Login: "octocat"})
			err1 := db.InsertToken(&common.Token{Login: "octocat", Label: "gist"})
			err2 := db.InsertToken(&common.Token{Login: "octocat", Label: "github"})
			g.Assert(err1).Equal(nil)
			g.Assert(err2).Equal(nil)

			list, err := db.GetUserTokens("octocat")
			g.Assert(err).Equal(nil)
			g.Assert(len(list)).Equal(2)
		})
		g.It("Should delete")
		g.It("Should insert")
		g.It("Should not insert if exists")
	})
}
