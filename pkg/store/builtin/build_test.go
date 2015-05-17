package builtin

import (
	common "github.com/drone/drone/pkg/types"
	"github.com/franela/goblin"
	"testing"
)

func TestBuildstore(t *testing.T) {
	db := mustConnectTest()
	bs := NewBuildstore(db)
	cs := NewCommitstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Buildstore", func() {

		// before each test we purge the package table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM builds")
			db.Exec("DELETE FROM commits")
		})

		g.It("Should update an existing build in the datastore", func() {
			buildList := []*common.Build{
				&common.Build{
					CommitID: 1,
					State:    "success",
					ExitCode: 0,
					Sequence: 1,
				},
				&common.Build{
					CommitID: 3,
					State:    "error",
					ExitCode: 1,
					Sequence: 2,
				},
			}
			//In order for buid to be populated,
			//The AddCommit command will insert builds
			//if the Commit.Builds array is populated
			//Add Commit.
			commit1 := common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: buildList,
			}
			//Add commit, build, retrieve 2nd, update it and recheck it.
			err1 := cs.AddCommit(&commit1)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(commit1.ID != 0).IsTrue()
			g.Assert(commit1.Sequence).Equal(1)
			//
			build, err2 := bs.Build(commit1.Builds[1].ID)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(build.ID).Equal(commit1.Builds[1].ID)
			build.State = common.StatePending
			err1 = bs.SetBuild(build)
			g.Assert(err1 == nil).IsTrue()
			build, err2 = bs.Build(commit1.Builds[1].ID)
			g.Assert(build.ID).Equal(commit1.Builds[1].ID)
			g.Assert(build.State).Equal(common.StatePending)
		})

		g.It("Should return a build by ID", func() {
			buildList := []*common.Build{
				&common.Build{
					CommitID: 1,
					State:    "success",
					ExitCode: 0,
					Sequence: 1,
				},
				&common.Build{
					CommitID: 3,
					State:    "error",
					ExitCode: 1,
					Sequence: 2,
				},
			}
			//In order for buid to be populated,
			//The AddCommit command will insert builds
			//if the Commit.Builds array is populated
			//Add Commit.
			commit1 := common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: buildList,
			}
			//Add commit, build, retrieve 2nd build ID.
			err1 := cs.AddCommit(&commit1)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(commit1.ID != 0).IsTrue()
			g.Assert(commit1.Sequence).Equal(1)
			//
			build, err2 := bs.Build(commit1.Builds[1].ID)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(build.ID).Equal(commit1.Builds[1].ID)
		})

		g.It("Should return a build by Sequence", func() {
			buildList := []*common.Build{
				&common.Build{
					CommitID: 1,
					State:    "success",
					ExitCode: 0,
					Sequence: 1,
				},
				&common.Build{
					CommitID: 3,
					State:    "error",
					ExitCode: 1,
					Sequence: 2,
				},
			}
			//In order for buid to be populated,
			//The AddCommit command will insert builds
			//if the Commit.Builds array is populated
			//Add Commit.
			commit1 := common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: buildList,
			}
			//Add commit, build, retrieve 2nd build ID.
			err1 := cs.AddCommit(&commit1)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(commit1.ID != 0).IsTrue()
			g.Assert(commit1.Sequence).Equal(1)
			//
			build, err2 := bs.BuildSeq(&commit1, commit1.Builds[1].Sequence)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(build.Sequence).Equal(commit1.Builds[1].Sequence)
		})

		g.It("Should return a list of all commit builds", func() {
			//Add repo
			buildList := []*common.Build{
				&common.Build{
					CommitID: 1,
					State:    "success",
					ExitCode: 0,
					Sequence: 1,
				},
				&common.Build{
					CommitID: 3,
					State:    "error",
					ExitCode: 1,
					Sequence: 2,
				},
				&common.Build{
					CommitID: 5,
					State:    "pending",
					ExitCode: 0,
					Sequence: 3,
				},
			}
			//In order for buid to be populated,
			//The AddCommit command will insert builds
			//if the Commit.Builds array is populated
			//Add Commit.
			commit1 := common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "14710626f22791619d3b7e9ccf58b10374e5b76d",
				Builds: buildList,
			}
			//
			err1 := cs.AddCommit(&commit1)
			g.Assert(err1 == nil).IsTrue()
			bldList, err2 := bs.BuildList(&commit1)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(len(bldList)).Equal(3)
			g.Assert(bldList[0].Sequence).Equal(1)
			g.Assert(bldList[0].State).Equal(common.StateSuccess)
		})
	})
}
