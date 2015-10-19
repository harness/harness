package model

import (
	"testing"

	"github.com/drone/drone/shared/database"
	"github.com/franela/goblin"
)

func TestRepostore(t *testing.T) {
	db := database.OpenTest()
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Repo", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM builds")
			db.Exec("DELETE FROM repos")
			db.Exec("DELETE FROM users")
		})

		g.It("Should Set a Repo", func() {
			repo := Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err1 := CreateRepo(db, &repo)
			err2 := UpdateRepo(db, &repo)
			getrepo, err3 := GetRepo(db, repo.ID)
			if err3 != nil {
				println("Get Repo Error")
				println(err3.Error())
			}
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
		})

		g.It("Should Add a Repo", func() {
			repo := Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err := CreateRepo(db, &repo)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID != 0).IsTrue()
		})

		g.It("Should Get a Repo by ID", func() {
			repo := Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			CreateRepo(db, &repo)
			getrepo, err := GetRepo(db, repo.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Get a Repo by Name", func() {
			repo := Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			CreateRepo(db, &repo)
			getrepo, err := GetRepoName(db, repo.Owner, repo.Name)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Get a Repo List by User", func() {
			repo1 := Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			repo2 := Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone-dart",
				Owner:    "bradrydzewski",
				Name:     "drone-dart",
			}
			CreateRepo(db, &repo1)
			CreateRepo(db, &repo2)
			CreateBuild(db, &Build{RepoID: repo1.ID, Author: "bradrydzewski"})
			CreateBuild(db, &Build{RepoID: repo1.ID, Author: "johnsmith"})
			repos, err := GetRepoList(db, &User{ID: 1, Login: "bradrydzewski"})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(repos)).Equal(1)
			g.Assert(repos[0].UserID).Equal(repo1.UserID)
			g.Assert(repos[0].Owner).Equal(repo1.Owner)
			g.Assert(repos[0].Name).Equal(repo1.Name)
		})

		g.It("Should Delete a Repo", func() {
			repo := Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			CreateRepo(db, &repo)
			_, err1 := GetRepo(db, repo.ID)
			err2 := DeleteRepo(db, &repo)
			_, err3 := GetRepo(db, repo.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should Enforce Unique Repo Name", func() {
			repo1 := Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			repo2 := Repo{
				UserID:   2,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err1 := CreateRepo(db, &repo1)
			err2 := CreateRepo(db, &repo2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})
	})
}
