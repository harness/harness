package builtin

import (
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	"github.com/drone/drone/pkg/types"
)

func TestCommitstore(t *testing.T) {
	db := mustConnectTest()
	bs := NewBuildstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Buildstore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM builds")
			db.Exec("DELETE FROM jobs")
		})

		g.It("Should Post a Build", func() {
			build := types.Build{
				RepoID: 1,
				Status: types.StateSuccess,
				Commit: &types.Commit{
					Ref: "refs/heads/master",
					Sha: "85f8c029b902ed9400bc600bac301a0aadb144ac",
				},
			}
			err := bs.AddBuild(&build)
			g.Assert(err == nil).IsTrue()
			g.Assert(build.ID != 0).IsTrue()
			g.Assert(build.Number).Equal(1)
			g.Assert(build.Commit.Ref).Equal("refs/heads/master")
			g.Assert(build.Commit.Sha).Equal("85f8c029b902ed9400bc600bac301a0aadb144ac")
		})

		g.It("Should Put a Build", func() {
			build := types.Build{
				RepoID: 1,
				Number: 5,
				Status: types.StatePending,
				Commit: &types.Commit{
					Ref: "refs/heads/master",
					Sha: "85f8c029b902ed9400bc600bac301a0aadb144ac",
				},
			}
			bs.AddBuild(&build)
			build.Status = types.StateRunning
			err1 := bs.SetBuild(&build)
			getbuild, err2 := bs.Build(build.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(build.ID).Equal(getbuild.ID)
			g.Assert(build.RepoID).Equal(getbuild.RepoID)
			g.Assert(build.Status).Equal(getbuild.Status)
			g.Assert(build.Number).Equal(getbuild.Number)
		})

		g.It("Should Get a Build", func() {
			build := types.Build{
				RepoID: 1,
				Status: types.StateSuccess,
			}
			bs.AddBuild(&build)
			getbuild, err := bs.Build(build.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(build.ID).Equal(getbuild.ID)
			g.Assert(build.RepoID).Equal(getbuild.RepoID)
			g.Assert(build.Status).Equal(getbuild.Status)
		})

		g.It("Should Get a Build by Number", func() {
			build1 := &types.Build{
				RepoID: 1,
				Status: types.StatePending,
			}
			build2 := &types.Build{
				RepoID: 1,
				Status: types.StatePending,
			}
			err1 := bs.AddBuild(build1)
			err2 := bs.AddBuild(build2)
			getbuild, err3 := bs.BuildNumber(&types.Repo{ID: 1}, build2.Number)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
		})

		g.It("Should Kill Pending or Started Builds", func() {
			build1 := &types.Build{
				RepoID: 1,
				Status: types.StateRunning,
			}
			build2 := &types.Build{
				RepoID: 1,
				Status: types.StatePending,
			}
			bs.AddBuild(build1)
			bs.AddBuild(build2)
			err1 := bs.KillBuilds()
			getbuild1, err2 := bs.Build(build1.ID)
			getbuild2, err3 := bs.Build(build2.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(getbuild1.Status).Equal(types.StateKilled)
			g.Assert(getbuild2.Status).Equal(types.StateKilled)
		})

		g.It("Should get recent Builds", func() {
			build1 := &types.Build{
				RepoID: 1,
				Status: types.StateFailure,
			}
			build2 := &types.Build{
				RepoID: 1,
				Status: types.StateSuccess,
			}
			bs.AddBuild(build1)
			bs.AddBuild(build2)
			builds, err := bs.BuildList(&types.Repo{ID: 1}, 20, 0)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(builds)).Equal(2)
			g.Assert(builds[0].ID).Equal(build2.ID)
			g.Assert(builds[0].RepoID).Equal(build2.RepoID)
			g.Assert(builds[0].Status).Equal(build2.Status)
		})
		//
		// g.It("Should get the last Commit", func() {
		// 	commit1 := &types.Commit{
		// 		RepoID: 1,
		// 		State:  types.StateFailure,
		// 		Branch: "master",
		// 		Ref:    "refs/heads/master",
		// 		Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
		// 	}
		// 	commit2 := &types.Commit{
		// 		RepoID: 1,
		// 		State:  types.StateFailure,
		// 		Branch: "master",
		// 		Ref:    "refs/heads/master",
		// 		Sha:    "8d6a233744a5dcacbf2605d4592a4bfe8b37320d",
		// 	}
		// 	commit3 := &types.Commit{
		// 		RepoID: 1,
		// 		State:  types.StateSuccess,
		// 		Branch: "dev",
		// 		Ref:    "refs/heads/dev",
		// 		Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
		// 	}
		// 	err1 := bs.AddCommit(commit1)
		// 	err2 := bs.AddCommit(commit2)
		// 	err3 := bs.AddCommit(commit3)
		// 	last, err4 := bs.CommitLast(&types.Repo{ID: 1}, "master")
		// 	g.Assert(err1 == nil).IsTrue()
		// 	g.Assert(err2 == nil).IsTrue()
		// 	g.Assert(err3 == nil).IsTrue()
		// 	g.Assert(err4 == nil).IsTrue()
		// 	g.Assert(last.ID).Equal(commit2.ID)
		// 	g.Assert(last.RepoID).Equal(commit2.RepoID)
		// 	g.Assert(last.Sequence).Equal(commit2.Sequence)
		// })
	})
}
