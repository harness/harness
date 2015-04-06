package database

import (
	"fmt"
	"testing"

	"github.com/drone/drone/shared/model"
	"github.com/franela/goblin"
)

func TestCommitstore(t *testing.T) {
	db := mustConnectTest()
	cs := NewCommitstore(db)
	rs := NewRepostore(db)
	ps := NewPermstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Commitstore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM perms")
			db.Exec("DELETE FROM repos")
			db.Exec("DELETE FROM commits")
		})

		g.It("Should Put a Commit", func() {
			commit := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			err := cs.PutCommit(&commit)
			g.Assert(err == nil).IsTrue()
			g.Assert(commit.ID != 0).IsTrue()
		})

		g.It("Should Post a Commit", func() {
			commit := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			err := cs.PostCommit(&commit)
			g.Assert(err == nil).IsTrue()
			g.Assert(commit.ID != 0).IsTrue()
		})

		g.It("Should Get a Commit by ID", func() {
			commit := model.Commit{
				RepoID:  1,
				Branch:  "foo",
				Sha:     "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status:  model.StatusSuccess,
				Created: 1398065343,
				Updated: 1398065344,
			}
			cs.PostCommit(&commit)
			getcommit, err := cs.GetCommit(commit.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(commit.ID).Equal(getcommit.ID)
			g.Assert(commit.RepoID).Equal(getcommit.RepoID)
			g.Assert(commit.Branch).Equal(getcommit.Branch)
			g.Assert(commit.Sha).Equal(getcommit.Sha)
			g.Assert(commit.Status).Equal(getcommit.Status)
			g.Assert(commit.Created).Equal(getcommit.Created)
			g.Assert(commit.Updated).Equal(getcommit.Updated)
		})

		g.It("Should Get the build number", func() {
			commit := model.Commit{
				RepoID:  1,
				Branch:  "foo",
				Sha:     "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status:  model.StatusSuccess,
				Created: 1398065343,
				Updated: 1398065344,
			}
			cs.PostCommit(&commit)
			bn, err := cs.GetBuildNumber(&commit)
			g.Assert(err == nil).IsTrue()
			g.Assert(bn).Equal(int64(1))
		})

		g.It("Should Delete a Commit", func() {
			commit := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			cs.PostCommit(&commit)
			_, err1 := cs.GetCommit(commit.ID)
			err2 := cs.DelCommit(&commit)
			_, err3 := cs.GetCommit(commit.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should Kill Pending or Started Commits", func() {
			commit1 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusEnqueue,
			}
			commit2 := model.Commit{
				RepoID: 1,
				Branch: "bar",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusEnqueue,
			}
			cs.PutCommit(&commit1)
			cs.PutCommit(&commit2)
			err := cs.KillCommits()
			g.Assert(err == nil).IsTrue()
			getcommit1, _ := cs.GetCommit(commit1.ID)
			getcommit2, _ := cs.GetCommit(commit1.ID)
			g.Assert(getcommit1.Status).Equal(model.StatusKilled)
			g.Assert(getcommit2.Status).Equal(model.StatusKilled)
		})

		g.It("Should Get a Commit by Sha", func() {
			commit := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			cs.PostCommit(&commit)
			getcommit, err := cs.GetCommitSha(&model.Repo{ID: 1}, commit.Branch, commit.Sha)
			g.Assert(err == nil).IsTrue()
			g.Assert(commit.ID).Equal(getcommit.ID)
			g.Assert(commit.RepoID).Equal(getcommit.RepoID)
			g.Assert(commit.Branch).Equal(getcommit.Branch)
			g.Assert(commit.Sha).Equal(getcommit.Sha)
		})

		g.It("Should get the last Commit by Branch", func() {
			commit1 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusFailure,
			}
			commit2 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "0a74b46d7d62b737b6906897f48dbeb72cfda222",
				Status: model.StatusSuccess,
			}
			cs.PutCommit(&commit1)
			cs.PutCommit(&commit2)
			lastcommit, err := cs.GetCommitLast(&model.Repo{ID: 1}, commit1.Branch)
			g.Assert(err == nil).IsTrue()
			g.Assert(commit2.ID).Equal(lastcommit.ID)
			g.Assert(commit2.RepoID).Equal(lastcommit.RepoID)
			g.Assert(commit2.Branch).Equal(lastcommit.Branch)
			g.Assert(commit2.Sha).Equal(lastcommit.Sha)
		})

		g.It("Should get the recent Commit List for a Repo", func() {
			commit1 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusFailure,
			}
			commit2 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "0a74b46d7d62b737b6906897f48dbeb72cfda222",
				Status: model.StatusSuccess,
			}
			cs.PutCommit(&commit1)
			cs.PutCommit(&commit2)
			commits, err := cs.GetCommitList(&model.Repo{ID: 1}, 20, 0)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(commits)).Equal(2)
			g.Assert(commits[0].ID).Equal(commit2.ID)
			g.Assert(commits[0].RepoID).Equal(commit2.RepoID)
			g.Assert(commits[0].Branch).Equal(commit2.Branch)
			g.Assert(commits[0].Sha).Equal(commit2.Sha)
		})

		g.It("Should get only one last Commit from Commit List for a Repo", func() {
			commit1 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusFailure,
			}
			commit2 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "0a74b46d7d62b737b6906897f48dbeb72cfda222",
				Status: model.StatusSuccess,
			}
			cs.PutCommit(&commit1)
			cs.PutCommit(&commit2)
			commits, err := cs.GetCommitList(&model.Repo{ID: 1}, 1, 1)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(commits)).Equal(1)
			g.Assert(commits[0].ID).Equal(commit1.ID)
			g.Assert(commits[0].RepoID).Equal(commit1.RepoID)
			g.Assert(commits[0].Branch).Equal(commit1.Branch)
			g.Assert(commits[0].Sha).Equal(commit1.Sha)
		})

		g.It("Should get the recent Commit List for a User", func() {
			repo1 := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			repo2 := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "drone",
				Name:   "drone",
			}
			repo3 := model.Repo{
				UserID: 2,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "droneio",
				Name:   "drone",
			}
			rs.PostRepo(&repo1)
			rs.PostRepo(&repo2)
			commit1 := model.Commit{
				RepoID: repo1.ID,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusFailure,
			}
			commit2 := model.Commit{
				RepoID: repo2.ID,
				Branch: "bar",
				Sha:    "0a74b46d7d62b737b6906897f48dbeb72cfda222",
				Status: model.StatusSuccess,
			}
			commit3 := model.Commit{
				RepoID: 99999,
				Branch: "baz",
				Sha:    "0a74b46d7d62b737b6906897f48dbeb72cfda222",
				Status: model.StatusSuccess,
			}
			commit4 := model.Commit{
				RepoID: repo2.ID,
				Branch: "bar",
				Sha:    "d923a61d8ad3d8d02db4fef0bf40a726bad0fc03",
				Status: model.StatusStarted,
			}
			commit5 := model.Commit{
				RepoID: repo3.ID,
				Branch: "bar",
				Sha:    "d923a61d8ad3d8d02db4fef0bf40a726bad0fc03",
				Status: model.StatusStarted,
			}
			cs.PostCommit(&commit1)
			cs.PostCommit(&commit2)
			cs.PostCommit(&commit3)
			cs.PostCommit(&commit4)
			cs.PostCommit(&commit5)
			perm1 := model.Perm{
				RepoID: repo1.ID,
				UserID: 1,
				Read:   true,
				Write:  true,
				Admin:  true,
			}
			perm2 := model.Perm{
				RepoID: repo2.ID,
				UserID: 1,
				Read:   true,
				Write:  true,
				Admin:  true,
			}
			ps.PostPerm(&perm1)
			ps.PostPerm(&perm2)
			commits, err := cs.GetCommitListUser(&model.User{ID: 1})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(commits)).Equal(2)
			g.Assert(commits[0].RepoID).Equal(commit1.RepoID)
			g.Assert(commits[0].Branch).Equal(commit1.Branch)
			g.Assert(commits[0].Sha).Equal(commit1.Sha)
			g.Assert(commits[1].Sha).Equal(commit4.Sha)
			g.Assert(commits[1].Status).Equal(commit4.Status)

			commits, err = cs.GetCommitListActivity(&model.User{ID: 1}, 20, 0)
			fmt.Println(commits)
			fmt.Println(err)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(commits)).Equal(3)
		})

		g.It("Should get only one last Commit List for a User", func() {
			repo1 := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			repo2 := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "drone",
				Name:   "drone",
			}
			rs.PostRepo(&repo1)
			rs.PostRepo(&repo2)
			commit1 := model.Commit{
				RepoID: repo1.ID,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusFailure,
			}
			commit2 := model.Commit{
				RepoID: repo2.ID,
				Branch: "bar",
				Sha:    "0a74b46d7d62b737b6906897f48dbeb72cfda222",
				Status: model.StatusSuccess,
			}
			cs.PostCommit(&commit1)
			cs.PostCommit(&commit2)
			perm1 := model.Perm{
				RepoID: repo1.ID,
				UserID: 1,
				Read:   true,
				Write:  true,
				Admin:  true,
			}
			perm2 := model.Perm{
				RepoID: repo2.ID,
				UserID: 1,
				Read:   true,
				Write:  true,
				Admin:  true,
			}
			ps.PostPerm(&perm1)
			ps.PostPerm(&perm2)
			commits, err := cs.GetCommitListActivity(&model.User{ID: 1}, 1, 1)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(commits)).Equal(1)
			g.Assert(commits[0].RepoID).Equal(commit2.RepoID)
			g.Assert(commits[0].Branch).Equal(commit2.Branch)
			g.Assert(commits[0].Sha).Equal(commit2.Sha)
			g.Assert(commits[0].Status).Equal(commit2.Status)
		})

		g.It("Should enforce unique Sha + Branch", func() {
			commit1 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusEnqueue,
			}
			commit2 := model.Commit{
				RepoID: 1,
				Branch: "foo",
				Sha:    "85f8c029b902ed9400bc600bac301a0aadb144ac",
				Status: model.StatusEnqueue,
			}
			err1 := cs.PutCommit(&commit1)
			err2 := cs.PutCommit(&commit2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})
	})
}
