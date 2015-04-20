package bolt

import (
	"github.com/drone/drone/common"
	"github.com/drone/drone/common/sshutil"
	. "github.com/franela/goblin"
	"testing"
	//. "github.com/smartystreets/goconvey/convey"
)

func TestRepo(t *testing.T) {
	g := Goblin(t)
	g.Describe("Repo", func() {
		testUser := "octocat"
		testRepo := "github.com/octopod/hq"
		var db *DB // Temp database
		testParamsIns := map[string]string{
			"A": "Alpha",
			//"B": "Beta",
			//"C": "Charlie",
			//"D": "Delta",
			//"E": "Echo",
		}
		testParamsUpd := map[string]string{
			"A": "Alpha-Upd",
			//"B": "Beta-Upd",
			//"C": "Charlie-Upd",
			//"D": "Delta-Upd",
			//"E": "Echo-Upd",
		}

		// create a new database before each unit
		// test and destroy afterwards.
		g.BeforeEach(func() {
			db = Must("/tmp/drone.test.db")
		})
		g.AfterEach(func() {
			os.Remove(db.Path())
		})

		g.It("Should set Repo", func() {
			err := db.SetRepo(testRepo)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get Repo", func() {
			repo, err := db.Repo(testRepo)
			g.Assert(repo).Equal(tesRepo)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get RepoList", func() {
			repos, err := db.RepoList(testUser)
			g.Assert(repos).NotEqual(nil)
			g.Assert(err).Equal(nil)
		})

		g.It("Should set Subscriber", func() {
			err := db.SetSubscriber(testUser, testRepo)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get Subscribed", func() {
			subscribed, err := db.Subscribed(testUser, testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(subscribed).Equal(true)
		})

		g.It("Should set RepoParams", func() {
			//err := db.SetRepoParams(testRepo, testParamsIns)
			err := db.SetRepoParams(testRepo, map[string]string{"A": "Alpha"})
			g.Assert(err).Equal(nil)
		})

		g.It("Should get RepoParams", func() {
			params, err := db.RepoParams(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(params["A"]).Equal("Alpha")
		})

		// we test again with same repo/user already existing
		// to see if it will return "ErrConflict"
		g.It("Should set SetRepoNotExists", func() {
			err := db.SetRepoNotExists(testUser, testRepo)
			g.Assert(err).NotEqual(nil)
			g.Assert(err).Equal(ErrConflict)
		})

		g.It("Should set SetRepoNotExists", func() {
			err := db.SetRepoNotExists(testUser, testRepo)
			g.Assert(err).NotEqual(nil)
			g.Assert(err).Equal(ErrConflict)
		})

		g.It("Should set RepoKeypair", func() {
			err := db.SetRepoKeypair(testRepo, &common.Keypair{Private: []byte("A"), Public: []byte("Alpha")})
			g.Assert(err).Equal(nil)
		})

		g.It("Should get RepoKeypair", func() {
			keypair, err := db.RepoKeypair(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(keypair.Public).Equal([]byte("A"))
		})

		g.It("Should del Subscriber", func() {
			err := db.DelSubscriber(testUser, testRepo)
			g.Assert(err).Equal(nil)
		})

		g.It("Should del Repo", func() {
			err := db.DelRepo(repo)
			g.Assert(err).Equal(nil)
		})
	})
}
