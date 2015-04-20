package bolt

import (
	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
	"os"
	"testing"
)

func TestRepo(t *testing.T) {
	g := Goblin(t)
	g.Describe("Repo", func() {
		testUser := "octocat"
		testRepo := "github.com/octopod/hq"
		testRepo2 := "github.com/octopod/avengers"
		var db *DB // Temp database

		// create a new database before each unit
		// test and destroy afterwards.
		g.BeforeEach(func() {
			db = Must("/tmp/drone.test.db")
		})
		g.AfterEach(func() {
			os.Remove(db.Path())
		})

		g.It("Should set Repo", func() {
			//err := db.SetRepoNotExists(&common.User{Name: testUser}, &common.Repo{Name: testRepo})
			err := db.SetRepo(&common.Repo{FullName: testRepo})
			g.Assert(err).Equal(nil)

			// setrepo only returns an error. Repo returns error and a structure
			repo, err := db.Repo(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(repo.FullName).Equal(testRepo)
		})

		g.It("Should get Repo", func() {
			//db.SetRepoNotExists(&common.User{Name: testUser}, &common.Repo{Name: testRepo})
			db.SetRepo(&common.Repo{FullName: testRepo})

			// setrepo only returns an error. Repo returns error and a structure
			repo, err := db.Repo(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(repo.FullName).Equal(testRepo)
		})

		g.It("Should del Repo", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			// setrepo only returns an error. Repo returns error and a structure
			//repo, err := db.Repo(testRepo)
			db.Repo(testRepo)
			err_ := db.DelRepo((&common.Repo{FullName: testRepo}))
			g.Assert(err_).Equal(nil)
		})

		g.It("Should get RepoList", func() {
			db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo})
			db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo2})
			//db.SetRepo(&common.Repo{FullName: testRepo})
			//db.SetRepo(&common.Repo{FullName: testRepo2})
			repos, err := db.RepoList(testUser)
			g.Assert(err).Equal(nil)
			g.Assert(len(repos)).Equal(2)
		})

		g.It("Should set RepoParams", func() {
			//db.SetRepoNotExists(&common.User{Name: testUser}, &common.Repo{Name: testRepo})
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetRepoParams(testRepo, map[string]string{"A": "Alpha"})
			g.Assert(err).Equal(nil)
		})

		g.It("Should get RepoParams", func() {
			//db.SetRepoNotExists(&common.User{Name: testUser}, &common.Repo{Name: testRepo})
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetRepoParams(testRepo, map[string]string{"A": "Alpha", "B": "Beta"})
			params, err := db.RepoParams(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(len(params)).Equal(2)
			g.Assert(params["A"]).Equal("Alpha")
			g.Assert(params["B"]).Equal("Beta")
		})

		// we test again with same repo/user already existing
		// to see if it will return "ErrConflict"
		g.It("Should set SetRepoNotExists", func() {
			err := db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo})
			g.Assert(err).Equal(nil)
			// We should get ErrConflict now, trying to add the same repo again.
			err_ := db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo})
			g.Assert(err_ == nil).IsFalse() // we should get (ErrConflict)
		})

		g.It("Should set RepoKeypair", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			//err := db.SetRepoKeypair(testRepo, &common.Keypair{Private: []byte("A"), Public: []byte("Alpha")})
			err := db.SetRepoKeypair(testRepo, &common.Keypair{Private: "A", Public: "Alpha"})
			g.Assert(err).Equal(nil)
		})

		g.It("Should get RepoKeypair", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetRepoKeypair(testRepo, &common.Keypair{Private: "A", Public: "Alpha"})
			//g.Assert(err).Equal(nil)
			keypair, err := db.RepoKeypair(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(keypair.Public).Equal("Alpha")
			g.Assert(keypair.Private).Equal("A")
		})

		g.It("Should set Subscriber", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetSubscriber(testUser, testRepo)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get Subscribed", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetSubscriber(testUser, testRepo)
			subscribed, err := db.Subscribed(testUser, testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(subscribed).Equal(true)
		})

		g.It("Should del Subscriber", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			db.SetSubscriber(testUser, testRepo)
			err := db.DelSubscriber(testUser, testRepo)
			g.Assert(err).Equal(nil)
			//
			subscribed, err := db.Subscribed(testUser, testRepo)
			g.Assert(subscribed).Equal(false)

		})

	})
}
