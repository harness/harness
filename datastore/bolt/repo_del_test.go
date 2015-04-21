package bolt

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/drone/drone/common"
	. "github.com/franela/goblin"
)

func TestRepoDel(t *testing.T) {
	g := Goblin(t)
	g.Describe("Delete repo", func() {

		var db *DB // temporary database

		user := &common.User{Login: "freya"}
		repoUri := string("github.com/octopod/hq")

		// create a new database before each unit
		// test and destroy afterwards.
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

		g.It("should cleanup", func() {
			repo := &common.Repo{FullName: repoUri}
			err := db.SetRepoNotExists(user, repo)
			g.Assert(err).Equal(nil)

			db.SetBuild(repoUri, &common.Build{State: "success"})
			db.SetBuild(repoUri, &common.Build{State: "success"})
			db.SetBuild(repoUri, &common.Build{State: "pending"})

			db.SetBuildStatus(repoUri, 1, &common.Status{Context: "success"})
			db.SetBuildStatus(repoUri, 2, &common.Status{Context: "success"})
			db.SetBuildStatus(repoUri, 3, &common.Status{Context: "pending"})

			// first a little sanity to validate our test conditions
			_, err = db.BuildLast(repoUri)
			g.Assert(err).Equal(nil)

			// now run our specific test suite
			// 1. ensure that we can delete the repo
			err = db.DelRepo(repo)
			g.Assert(err).Equal(nil)

			// 2. ensure that deleting the repo cleans up other references
			_, err = db.Build(repoUri, 1)
			g.Assert(err).Equal(ErrKeyNotFound)
		})
	})

}
