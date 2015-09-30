package model

import (
	"testing"

	"github.com/drone/drone/shared/database"
	"github.com/franela/goblin"
)

func TestStarstore(t *testing.T) {
	db := database.Open("sqlite3", ":memory:")
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Stars", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM stars")
		})

		g.It("Should Add a Star", func() {
			user := User{ID: 1}
			repo := Repo{ID: 2}
			err := CreateStar(db, &user, &repo)
			g.Assert(err == nil).IsTrue()
		})

		g.It("Should Get Starred", func() {
			user := User{ID: 1}
			repo := Repo{ID: 2}
			CreateStar(db, &user, &repo)
			ok, err := GetStar(db, &user, &repo)
			g.Assert(err == nil).IsTrue()
			g.Assert(ok).IsTrue()
		})

		g.It("Should Not Get Starred", func() {
			user := User{ID: 1}
			repo := Repo{ID: 2}
			ok, err := GetStar(db, &user, &repo)
			g.Assert(err != nil).IsTrue()
			g.Assert(ok).IsFalse()
		})

		g.It("Should Del a Star", func() {
			user := User{ID: 1}
			repo := Repo{ID: 2}
			CreateStar(db, &user, &repo)
			_, err1 := GetStar(db, &user, &repo)
			err2 := DeleteStar(db, &user, &repo)
			_, err3 := GetStar(db, &user, &repo)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})
	})
}
