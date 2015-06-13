package builtin

import (
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	common "github.com/drone/drone/pkg/types"
)

func TestRepostore(t *testing.T) {
	db := mustConnectTest()
	rs := NewRepostore(db)
	ss := NewStarstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Repostore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM stars")
			db.Exec("DELETE FROM repos")
			db.Exec("DELETE FROM users")
		})

		g.It("Should Set a Repo", func() {
			repo := common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			err1 := rs.AddRepo(&repo)
			err2 := rs.SetRepo(&repo)
			getrepo, err3 := rs.Repo(repo.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
		})

		g.It("Should Add a Repo", func() {
			repo := common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			err := rs.AddRepo(&repo)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID != 0).IsTrue()
		})

		g.It("Should Get a Repo by ID", func() {
			repo := common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			rs.AddRepo(&repo)
			getrepo, err := rs.Repo(repo.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Get a Repo by Name", func() {
			repo := common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			rs.AddRepo(&repo)
			getrepo, err := rs.RepoName(repo.Owner, repo.Name)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Get a Repo List by User", func() {
			repo1 := common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			repo2 := common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone-dart",
			}
			rs.AddRepo(&repo1)
			rs.AddRepo(&repo2)
			ss.AddStar(&common.User{ID: 1}, &repo1)
			repos, err := rs.RepoList(&common.User{ID: 1})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(repos)).Equal(1)
			g.Assert(repos[0].UserID).Equal(repo1.UserID)
			g.Assert(repos[0].Owner).Equal(repo1.Owner)
			g.Assert(repos[0].Name).Equal(repo1.Name)
		})

		g.It("Should Delete a Repo", func() {
			repo := common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			rs.AddRepo(&repo)
			_, err1 := rs.Repo(repo.ID)
			err2 := rs.DelRepo(&repo)
			_, err3 := rs.Repo(repo.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should Enforce Unique Repo Name", func() {
			repo1 := common.Repo{
				UserID: 1,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			repo2 := common.Repo{
				UserID: 2,
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			err1 := rs.AddRepo(&repo1)
			err2 := rs.AddRepo(&repo2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})
	})
}
