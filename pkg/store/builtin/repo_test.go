package builtin

import (
	"testing"

	common "github.com/drone/drone/pkg/types"
	"github.com/franela/goblin"
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

		// g.It("Should Add a Repos Keypair", func() {
		// 	keypair := common.Keypair{
		// 		RepoID:  1,
		// 		Public:  []byte("-----BEGIN RSA PRIVATE KEY----- ..."),
		// 		Private: []byte("ssh-rsa AAAAE1BzbF1xc2EABAvVA6Z ..."),
		// 	}
		// 	err := rs.SetRepoKeypair(&keypair)
		// 	g.Assert(err == nil).IsTrue()
		// 	g.Assert(keypair.ID != 0).IsTrue()
		// 	getkeypair, err := rs.RepoKeypair(&common.Repo{ID: 1})
		// 	g.Assert(err == nil).IsTrue()
		// 	g.Assert(keypair.ID).Equal(getkeypair.ID)
		// 	g.Assert(keypair.RepoID).Equal(getkeypair.RepoID)
		// 	g.Assert(keypair.Public).Equal(getkeypair.Public)
		// 	g.Assert(keypair.Private).Equal(getkeypair.Private)
		// })

		// g.It("Should Add a Repos Private Params", func() {
		// 	params := common.Params{
		// 		RepoID: 1,
		// 		Map:    map[string]string{"foo": "bar"},
		// 	}
		// 	err := rs.SetRepoParams(&params)
		// 	g.Assert(err == nil).IsTrue()
		// 	g.Assert(params.ID != 0).IsTrue()
		// 	getparams, err := rs.RepoParams(&common.Repo{ID: 1})
		// 	g.Assert(err == nil).IsTrue()
		// 	g.Assert(params.ID).Equal(getparams.ID)
		// 	g.Assert(params.RepoID).Equal(getparams.RepoID)
		// 	g.Assert(params.Map).Equal(getparams.Map)
		// })

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
