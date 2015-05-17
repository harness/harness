package builtin

import (
	"testing"

	common "github.com/drone/drone/pkg/types"
	"github.com/franela/goblin"
)

func TestStarstore(t *testing.T) {
	db := mustConnectTest()
	ss := NewStarstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Starstore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM stars")
		})

		g.It("Should Add a Star", func() {
			user := common.User{ID: 1}
			repo := common.Repo{ID: 2}
			err := ss.AddStar(&user, &repo)
			g.Assert(err == nil).IsTrue()
		})

		g.It("Should Get Starred", func() {
			user := common.User{ID: 1}
			repo := common.Repo{ID: 2}
			ss.AddStar(&user, &repo)
			ok, err := ss.Starred(&user, &repo)
			g.Assert(err == nil).IsTrue()
			g.Assert(ok).IsTrue()
		})

		g.It("Should Not Get Starred", func() {
			user := common.User{ID: 1}
			repo := common.Repo{ID: 2}
			ok, err := ss.Starred(&user, &repo)
			g.Assert(err != nil).IsTrue()
			g.Assert(ok).IsFalse()
		})

		g.It("Should Del a Star", func() {
			user := common.User{ID: 1}
			repo := common.Repo{ID: 2}
			ss.AddStar(&user, &repo)
			_, err1 := ss.Starred(&user, &repo)
			err2 := ss.DelStar(&user, &repo)
			_, err3 := ss.Starred(&user, &repo)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})
	})
}
