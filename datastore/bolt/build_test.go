package bolt

import (
	"os"
	"testing"

	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
)

func TestBuild(t *testing.T) {
	g := Goblin(t)
	g.Describe("Build", func() {
		var db *DB // temporary database
		repo := string("github.com/octopod/hq")

		// create a new database before each unit
		// test and destroy afterwards.
		g.BeforeEach(func() {
			db = Must("/tmp/drone.test.db")
		})
		g.AfterEach(func() {
			os.Remove(db.Path())
		})

		g.It("Should sequence builds", func() {
			err := db.InsertBuild(repo, &common.Build{State: "pending"})
			g.Assert(err).Equal(nil)

			// the first build should always be numero 1
			build, err := db.GetBuild(repo, 1)
			g.Assert(err).Equal(nil)
			g.Assert(build.State).Equal("pending")

			// add another build, just for fun
			err = db.InsertBuild(repo, &common.Build{State: "success"})
			g.Assert(err).Equal(nil)

			// get the next build
			build, err = db.GetBuild(repo, 2)
			g.Assert(err).Equal(nil)
			g.Assert(build.State).Equal("success")
		})

		g.It("Should get the latest builds", func() {
			db.InsertBuild(repo, &common.Build{State: "success"})
			db.InsertBuild(repo, &common.Build{State: "success"})
			db.InsertBuild(repo, &common.Build{State: "pending"})

			build, err := db.GetBuildLast(repo)
			g.Assert(err).Equal(nil)
			g.Assert(build.State).Equal("pending")
			g.Assert(build.Number).Equal(3)
		})
	})
}
