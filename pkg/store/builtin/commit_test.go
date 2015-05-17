package builtin

import (
	"testing"

	common "github.com/drone/drone/pkg/types"
	"github.com/franela/goblin"
)

func TestCommitstore(t *testing.T) {
	db := mustConnectTest()
	bs := NewCommitstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Commitstore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM commits")
			db.Exec("DELETE FROM tasks")
		})

		g.It("Should Post a Commit", func() {
			commit := common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/master",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			err := bs.AddCommit(&commit)
			g.Assert(err == nil).IsTrue()
			g.Assert(commit.ID != 0).IsTrue()
			g.Assert(commit.Sequence).Equal(1)
		})

		g.It("Should Put a Commit", func() {
			commit := common.Commit{
				RepoID:   1,
				Sequence: 5,
				State:    common.StatePending,
				Ref:      "refs/heads/master",
				Sha:      "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			bs.AddCommit(&commit)
			commit.State = common.StateRunning
			err1 := bs.SetCommit(&commit)
			getcommit, err2 := bs.Commit(commit.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(commit.ID).Equal(getcommit.ID)
			g.Assert(commit.RepoID).Equal(getcommit.RepoID)
			g.Assert(commit.State).Equal(getcommit.State)
			g.Assert(commit.Sequence).Equal(getcommit.Sequence)
		})

		g.It("Should Get a Commit", func() {
			commit := common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
			}
			bs.AddCommit(&commit)
			getcommit, err := bs.Commit(commit.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(commit.ID).Equal(getcommit.ID)
			g.Assert(commit.RepoID).Equal(getcommit.RepoID)
			g.Assert(commit.State).Equal(getcommit.State)
		})

		g.It("Should Get a Commit by Sequence", func() {
			commit1 := &common.Commit{
				RepoID: 1,
				State:  common.StatePending,
				Ref:    "refs/heads/master",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			commit2 := &common.Commit{
				RepoID: 1,
				State:  common.StatePending,
				Ref:    "refs/heads/dev",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			err1 := bs.AddCommit(commit1)
			err2 := bs.AddCommit(commit2)
			getcommit, err3 := bs.CommitSeq(&common.Repo{ID: 1}, commit2.Sequence)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(commit2.ID).Equal(getcommit.ID)
			g.Assert(commit2.RepoID).Equal(getcommit.RepoID)
			g.Assert(commit2.Sequence).Equal(getcommit.Sequence)
		})

		g.It("Should Kill Pending or Started Commits", func() {
			commit1 := &common.Commit{
				RepoID: 1,
				State:  common.StateRunning,
				Ref:    "refs/heads/master",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			commit2 := &common.Commit{
				RepoID: 1,
				State:  common.StatePending,
				Ref:    "refs/heads/dev",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			bs.AddCommit(commit1)
			bs.AddCommit(commit2)
			err1 := bs.KillCommits()
			getcommit1, err2 := bs.Commit(commit1.ID)
			getcommit2, err3 := bs.Commit(commit2.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(getcommit1.State).Equal(common.StateKilled)
			g.Assert(getcommit2.State).Equal(common.StateKilled)
		})

		g.It("Should get recent Commits", func() {
			commit1 := &common.Commit{
				RepoID: 1,
				State:  common.StateFailure,
				Ref:    "refs/heads/master",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			commit2 := &common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Ref:    "refs/heads/dev",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			bs.AddCommit(commit1)
			bs.AddCommit(commit2)
			commits, err := bs.CommitList(&common.Repo{ID: 1}, 20, 0)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(commits)).Equal(2)
			g.Assert(commits[0].ID).Equal(commit2.ID)
			g.Assert(commits[0].RepoID).Equal(commit2.RepoID)
			g.Assert(commits[0].State).Equal(commit2.State)
		})

		g.It("Should get the last Commit", func() {
			commit1 := &common.Commit{
				RepoID: 1,
				State:  common.StateFailure,
				Branch: "master",
				Ref:    "refs/heads/master",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			commit2 := &common.Commit{
				RepoID: 1,
				State:  common.StateFailure,
				Branch: "master",
				Ref:    "refs/heads/master",
				Sha:    "8d6a233744a5dcacbf2605d4592a4bfe8b37320d",
			}
			commit3 := &common.Commit{
				RepoID: 1,
				State:  common.StateSuccess,
				Branch: "dev",
				Ref:    "refs/heads/dev",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			err1 := bs.AddCommit(commit1)
			err2 := bs.AddCommit(commit2)
			err3 := bs.AddCommit(commit3)
			last, err4 := bs.CommitLast(&common.Repo{ID: 1}, "master")
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(err4 == nil).IsTrue()
			g.Assert(last.ID).Equal(commit2.ID)
			g.Assert(last.RepoID).Equal(commit2.RepoID)
			g.Assert(last.Sequence).Equal(commit2.Sequence)
		})
	})
}
