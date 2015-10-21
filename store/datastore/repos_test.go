package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func Test_repostore(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
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
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err1 := s.Repos().Create(&repo)
			err2 := s.Repos().Update(&repo)
			getrepo, err3 := s.Repos().Get(repo.ID)
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
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err := s.Repos().Create(&repo)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID != 0).IsTrue()
		})

		g.It("Should Get a Repo by ID", func() {
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			s.Repos().Create(&repo)
			getrepo, err := s.Repos().Get(repo.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		g.It("Should Get a Repo by Name", func() {
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			s.Repos().Create(&repo)
			getrepo, err := s.Repos().GetName(repo.FullName)
			g.Assert(err == nil).IsTrue()
			g.Assert(repo.ID).Equal(getrepo.ID)
			g.Assert(repo.UserID).Equal(getrepo.UserID)
			g.Assert(repo.Owner).Equal(getrepo.Owner)
			g.Assert(repo.Name).Equal(getrepo.Name)
		})

		// g.It("Should Get a Repo List by User", func() {
		// 	repo1 := model.Repo{
		// 		UserID:   1,
		// 		FullName: "bradrydzewski/drone",
		// 		Owner:    "bradrydzewski",
		// 		Name:     "drone",
		// 	}
		// 	repo2 := model.Repo{
		// 		UserID:   1,
		// 		FullName: "bradrydzewski/drone-dart",
		// 		Owner:    "bradrydzewski",
		// 		Name:     "drone-dart",
		// 	}
		// 	s.Repos().Create(&repo1)
		// 	s.Repos().Create(&repo2)
		// 	CreateBuild(db, &Build{RepoID: repo1.ID, Author: "bradrydzewski"})
		// 	CreateBuild(db, &Build{RepoID: repo1.ID, Author: "johnsmith"})
		// 	repos, err := GetRepoList(db, &User{ID: 1, Login: "bradrydzewski"})
		// 	g.Assert(err == nil).IsTrue()
		// 	g.Assert(len(repos)).Equal(1)
		// 	g.Assert(repos[0].UserID).Equal(repo1.UserID)
		// 	g.Assert(repos[0].Owner).Equal(repo1.Owner)
		// 	g.Assert(repos[0].Name).Equal(repo1.Name)
		// })

		g.It("Should Delete a Repo", func() {
			repo := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			s.Repos().Create(&repo)
			_, err1 := s.Repos().Get(repo.ID)
			err2 := s.Repos().Delete(&repo)
			_, err3 := s.Repos().Get(repo.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})

		g.It("Should Enforce Unique Repo Name", func() {
			repo1 := model.Repo{
				UserID:   1,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			repo2 := model.Repo{
				UserID:   2,
				FullName: "bradrydzewski/drone",
				Owner:    "bradrydzewski",
				Name:     "drone",
			}
			err1 := s.Repos().Create(&repo1)
			err2 := s.Repos().Create(&repo2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})
	})
}
