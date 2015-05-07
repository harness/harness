package builtin

import (
	"bytes"
	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestRepo(t *testing.T) {
	g := Goblin(t)
	g.Describe("Repo", func() {
		testUser := "octocat"
		testRepo := "github.com/octopod/hq"
		testRepo2 := "github.com/octopod/avengers"
		commUser := &common.User{Login: "freya"}
		var db *DB // Temp database

		// create a new database before each unit test and destroy afterwards.
		g.BeforeEach(func() {
			file, err := ioutil.TempFile(os.TempDir(), "drone-bolt")
			if err != nil {
				panic(err)
			}

			db = Must(file.Name())
		})
		g.AfterEach(func() {
			os.Remove(db.Path())
		})

		g.It("Should set Repo", func() {
			err := db.SetRepo(&common.Repo{FullName: testRepo})
			g.Assert(err).Equal(nil)

			repo, err := db.Repo(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(repo.FullName).Equal(testRepo)
		})

		g.It("Should get Repo", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})

			repo, err := db.Repo(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(repo.FullName).Equal(testRepo)
		})

		g.It("Should be deletable", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})

			db.Repo(testRepo)
			err_ := db.DelRepo((&common.Repo{FullName: testRepo}))
			g.Assert(err_).Equal(nil)
		})

		g.It("Should cleanup builds when deleted", func() {
			repo := &common.Repo{FullName: testRepo}
			err := db.SetRepoNotExists(commUser, repo)
			g.Assert(err).Equal(nil)

			db.SetBuild(testRepo, &common.Build{State: "success"})
			db.SetBuild(testRepo, &common.Build{State: "success"})
			db.SetBuild(testRepo, &common.Build{State: "pending"})
			//db.SetLogs(testRepo, 1, 1, []byte("foo"))
			db.SetLogs(testRepo, 1, 1, io.Reader(bytes.NewBuffer([]byte("foo"))))

			// first a little sanity to validate our test conditions
			_, err = db.BuildLast(testRepo)
			g.Assert(err).Equal(nil)

			// now run our specific test suite
			// 1. ensure that we can delete the repo
			err = db.DelRepo(repo)
			g.Assert(err).Equal(nil)

			// 2. ensure that deleting the repo cleans up other references
			_, err = db.Build(testRepo, 1)
			g.Assert(err).Equal(ErrKeyNotFound)
		})

		g.It("Should get Repo list", func() {
			db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo})
			db.SetRepoNotExists(&common.User{Login: testUser}, &common.Repo{FullName: testRepo2})

			repos, err := db.RepoList(testUser)
			g.Assert(err).Equal(nil)
			g.Assert(len(repos)).Equal(2)
		})

		g.It("Should set Repo parameters", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetRepoParams(testRepo, map[string]string{"A": "Alpha"})
			g.Assert(err).Equal(nil)
		})

		g.It("Should get Repo parameters", func() {
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
			g.Assert(err_).Equal(ErrKeyExists)
		})

		g.It("Should set Repo keypair", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})

			err := db.SetRepoKeypair(testRepo, &common.Keypair{Private: "A", Public: "Alpha"})
			g.Assert(err).Equal(nil)
		})

		g.It("Should get Repo keypair", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetRepoKeypair(testRepo, &common.Keypair{Private: "A", Public: "Alpha"})

			keypair, err := db.RepoKeypair(testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(keypair.Public).Equal("Alpha")
			g.Assert(keypair.Private).Equal("A")
		})

		g.It("Should set subscriber", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetSubscriber(testUser, testRepo)
			g.Assert(err).Equal(nil)
		})

		g.It("Should get subscribed", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			err := db.SetSubscriber(testUser, testRepo)
			subscribed, err := db.Subscribed(testUser, testRepo)
			g.Assert(err).Equal(nil)
			g.Assert(subscribed).Equal(true)
		})

		g.It("Should del subscriber", func() {
			db.SetRepo(&common.Repo{FullName: testRepo})
			db.SetSubscriber(testUser, testRepo)
			err := db.DelSubscriber(testUser, testRepo)
			g.Assert(err).Equal(nil)

			subscribed, err := db.Subscribed(testUser, testRepo)
			g.Assert(subscribed).Equal(false)

		})

	})
}
