package builtin

import (
	"github.com/drone/drone/common"
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

		g.It("NewBuildstore()", func() {
			//Add build
			build := common.Build{
				ID:       1,
				CommitID: 1,
				State:    "success",
			}
			err1 := bs.SetBuild(&build)
			g.Assert(err1 == nil).IsTrue()
		})

		g.It("Build()", func() {
			build_list := []*common.Build{
				&common.Build{
					//ID:       1,
					CommitID: 1,
					State:    "success",
					ExitCode: 0,
					Sequence: 1,
				},
				&common.Build{
					//ID:       2,
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
				Builds: build_list,
			}
			//
			err1 := cs.AddCommit(&commit1)
			g.Assert(err1 == nil).IsTrue()
			build, err2 := bs.Build(1)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(build.ID == 1).IsTrue()
		})

		g.It("BuildSeq()", func() {
			build_list := []*common.Build{
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
				Builds: build_list,
			}
			//
			err1 := cs.AddCommit(&commit1)
			g.Assert(err1 == nil).IsTrue()
			build, err2 := bs.BuildSeq(&commit1, 2)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(build.Sequence == 2).IsTrue()
		})

		g.It("BuildList()", func() {
			//Add repo
			build_list := []*common.Build{
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
				Builds: build_list,
			}
			//
			err1 := cs.AddCommit(&commit1)
			g.Assert(err1 == nil).IsTrue()
			buildList, err2 := bs.BuildList(&commit1)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(len(buildList)).Equal(3)
			g.Assert(build_list[0].Sequence).Equal(1)
			g.Assert(build_list[0].State).Equal(common.StateSuccess)
		})
	})
}
