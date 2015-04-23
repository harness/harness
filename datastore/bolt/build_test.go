package bolt

import (
	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
	"os"
	"testing"
)

func TestBuild(t *testing.T) {
	g := Goblin(t)
	g.Describe("Build", func() {
		var db *DB // temporary database
		repo := string("github.com/octopod/hq")
		//testUser := &common.User{Login: "octocat"}
		//testRepo := &common.Repo{FullName: "github.com/octopod/hq"}
		testUser := "octocat"
		testRepo := "github.com/octopod/hq"
		//testBuild := 1

		// create a new database before each unit
		// test and destroy afterwards.
		g.BeforeEach(func() {
			db = Must("/tmp/drone.test.db")
		})
		g.AfterEach(func() {
			os.Remove(db.Path())
		})

		g.It("Should sequence builds", func() {
			err := db.SetBuild(repo, &common.Build{State: "pending"})
			g.Assert(err).Equal(nil)

			// the first build should always be numero 1
			build, err := db.Build(repo, 1)
			g.Assert(err).Equal(nil)
			g.Assert(build.State).Equal("pending")

			// add another build, just for fun
			err = db.SetBuild(repo, &common.Build{State: "success"})
			g.Assert(err).Equal(nil)

			// get the next build
			build, err = db.Build(repo, 2)
			g.Assert(err).Equal(nil)
			g.Assert(build.State).Equal("success")
		})

		g.It("Should get the latest builds", func() {
			db.SetBuild(repo, &common.Build{State: "success"})
			db.SetBuild(repo, &common.Build{State: "success"})
			db.SetBuild(repo, &common.Build{State: "pending"})

			build, err := db.BuildLast(repo)
			g.Assert(err).Equal(nil)
			g.Assert(build.State).Equal("pending")
			g.Assert(build.Number).Equal(3)
		})

		g.It("Should get the recent list of builds", func() {
			db.SetBuild(repo, &common.Build{State: "success"})
			db.SetBuild(repo, &common.Build{State: "success"})
			db.SetBuild(repo, &common.Build{State: "pending"})

			builds, err := db.BuildList(repo)
			g.Assert(err).Equal(nil)
			g.Assert(len(builds)).Equal(3)
		})

		g.It("Should set build status: SetBuildStatus()", func() {
			//err := db.SetRepoNotExists(testUser, testRepo)
			err := db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo})
			g.Assert(err).Equal(nil)

			db.SetBuild(repo, &common.Build{State: "error"})
			db.SetBuild(repo, &common.Build{State: "pending"})
			db.SetBuild(repo, &common.Build{State: "success"})
			err_ := db.SetBuildStatus(repo, 1, &common.Status{Context: "pending"})
			g.Assert(err_).Equal(nil)
			err_ = db.SetBuildStatus(repo, 2, &common.Status{Context: "running"})
			g.Assert(err_).Equal(nil)
			err_ = db.SetBuildStatus(repo, 3, &common.Status{Context: "success"})
			g.Assert(err_).Equal(nil)
		})

		g.It("Should set build state: SetBuildState()", func() {
			err := db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo})
			g.Assert(err).Equal(nil)

			db.SetBuild(repo, &common.Build{State: "error"})
			db.SetBuild(repo, &common.Build{State: "pending"})
			db.SetBuild(repo, &common.Build{State: "success"})
			err_ := db.SetBuildState(repo, &common.Build{Number: 1})
			g.Assert(err_).Equal(nil)
			err_ = db.SetBuildState(repo, &common.Build{Number: 2})
			g.Assert(err_).Equal(nil)
			err_ = db.SetBuildState(repo, &common.Build{Number: 3})
			g.Assert(err_).Equal(nil)
		})

		g.It("Should set build task: SetBuildTask()", func() {
			err := db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo})
			g.Assert(err).Equal(nil)

			db.SetBuild(repo, &common.Build{State: "error"})
			db.SetBuild(repo, &common.Build{State: "pending"})
			db.SetBuild(repo, &common.Build{State: "success"})
			err_ := db.SetBuildTask(repo, 1, &common.Task{Number: 1})
			g.Assert(err_).Equal(nil)
		})
	})
}
