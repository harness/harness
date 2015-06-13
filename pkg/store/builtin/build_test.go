package builtin

import (
	"testing"

	"github.com/bradrydzewski/drone/common"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	"github.com/drone/drone/pkg/types"
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

		g.It("Should Set a build", func() {
			build := &types.Build{
				CommitID: 1,
				State:    "pending",
				ExitCode: 0,
				Sequence: 1,
			}
			err1 := bs.AddBuild(build)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(build.ID != 0).IsTrue()

			build.State = "started"
			err2 := bs.SetBuild(build)
			g.Assert(err2 == nil).IsTrue()

			getbuild, err3 := bs.Build(build.ID)
			g.Assert(err3 == nil).IsTrue()
			g.Assert(getbuild.State).Equal(build.State)
		})

		g.It("Should Get a Build by ID", func() {
			build := &types.Build{
				CommitID:    1,
				State:       "pending",
				ExitCode:    1,
				Sequence:    1,
				Environment: map[string]string{"foo": "bar"},
			}
			err1 := bs.AddBuild(build)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(build.ID != 0).IsTrue()

			getbuild, err2 := bs.Build(build.ID)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(getbuild.ID).Equal(build.ID)
			g.Assert(getbuild.State).Equal(build.State)
			g.Assert(getbuild.ExitCode).Equal(build.ExitCode)
			g.Assert(getbuild.Environment).Equal(build.Environment)
			g.Assert(getbuild.Environment["foo"]).Equal("bar")
		})

		g.It("Should Get a Build by Sequence", func() {
			build := &types.Build{
				CommitID: 1,
				State:    "pending",
				ExitCode: 1,
				Sequence: 1,
			}
			err1 := bs.AddBuild(build)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(build.ID != 0).IsTrue()

			getbuild, err2 := bs.BuildSeq(&types.Commit{ID: 1}, 1)
			g.Assert(err2 == nil).IsTrue()
			g.Assert(getbuild.ID).Equal(build.ID)
			g.Assert(getbuild.State).Equal(build.State)
		})

		g.It("Should Get a List of Builds by Commit", func() {
			//Add repo
			buildList := []*types.Build{
				&types.Build{
					CommitID: 1,
					State:    "success",
					ExitCode: 0,
					Sequence: 1,
				},
				&types.Build{
					CommitID: 3,
					State:    "error",
					ExitCode: 1,
					Sequence: 2,
				},
				&types.Build{
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
			commit1 := types.Commit{
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
