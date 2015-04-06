package database

import (
	"testing"

	"github.com/drone/drone/shared/model"
	"github.com/franela/goblin"
)

func TestRepostore(t *testing.T) {
	db := mustConnectTest()
	rs := NewRepostore(db)
	ps := NewPermstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Repostore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM perms")
			db.Exec("DELETE FROM repos")
			db.Exec("DELETE FROM users")
		})

		g.It("Should Put a Repo", func() {
			repo := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			err := rs.PostRepo(&repo)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID != 0).IsTrue()
		})

		g.It("Should Post a Repo", func() {
			repo := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			err := rs.PostRepo(&repo)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID != 0).IsTrue()
		})

		g.It("Should Get a Repo by ID", func() {
			repo := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			rs.PostRepo(&repo)
			getrepo, err := rs.GetRepo(repo.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Remote).Equal(getrepo.Remote)
			g.Assert(repo.Host).Equal(getrepo.Host)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Get a Repo by Name", func() {
			repo := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			rs.PostRepo(&repo)
			getrepo, err := rs.GetRepoName(repo.Host, repo.Owner, repo.Name)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Remote).Equal(getrepo.Remote)
			g.Assert(repo.Host).Equal(getrepo.Host)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Get a Repo List by User", func() {
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
				Owner:  "bradrydzewski",
				Name:   "drone-dart",
			}
			rs.PostRepo(&repo1)
			rs.PostRepo(&repo2)
			ps.PostPerm(&model.Perm{
				RepoID: repo1.ID,
				UserID: 1,
				Read:   true,
				Write:  true,
				Admin:  true,
			})
			repos, err := rs.GetRepoList(&model.User{ID: 1})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(repos)).Equal(1)
			g.Assert(repos[0].UserID).Equal(repo1.UserID)
			g.Assert(repos[0].Remote).Equal(repo1.Remote)
			g.Assert(repos[0].Host).Equal(repo1.Host)
			g.Assert(repos[0].Owner).Equal(repo1.Owner)
			g.Assert(repos[0].Name).Equal(repo1.Name)
		})

		g.It("Should Delete a Repo", func() {
			repo := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			rs.PostRepo(&repo)
			_, err1 := rs.GetRepo(repo.ID)
			err2 := rs.DelRepo(&repo)
			_, err3 := rs.GetRepo(repo.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should Enforce Unique Repo Name", func() {
			repo1 := model.Repo{
				UserID: 1,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			repo2 := model.Repo{
				UserID: 2,
				Remote: "enterprise.github.com",
				Host:   "github.drone.io",
				Owner:  "bradrydzewski",
				Name:   "drone",
			}
			err1 := rs.PostRepo(&repo1)
			err2 := rs.PostRepo(&repo2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})
	})
}
