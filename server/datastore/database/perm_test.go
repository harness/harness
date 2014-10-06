package database

import (
	"testing"

	"github.com/drone/drone/shared/model"
	"github.com/franela/goblin"
)

func TestPermstore(t *testing.T) {
	db := mustConnectTest()
	ps := NewPermstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Permstore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM perms")
		})

		g.It("Should Put a Perm", func() {
			perm1 := model.Perm{
				UserID: 1,
				RepoID: 2,
				Read:   true,
				Write:  true,
				Admin:  true,
			}
			err := ps.PutPerm(&perm1)
			g.Assert(err == nil).IsTrue()
			g.Assert(perm1.ID != 0).IsTrue()
		})

		g.It("Should Post a Perm", func() {
			perm1 := model.Perm{
				UserID: 1,
				RepoID: 2,
				Read:   true,
				Write:  true,
				Admin:  true,
			}
			err := ps.PostPerm(&perm1)
			g.Assert(err == nil).IsTrue()
			g.Assert(perm1.ID != 0).IsTrue()
		})

		g.It("Should Upsert a Perm", func() {
			perm1 := model.Perm{
				UserID: 1,
				RepoID: 2,
				Read:   true,
				Write:  true,
				Admin:  true,
			}
			ps.PostPerm(&perm1)
			perm1.Read = true
			perm1.Write = true
			perm1.Admin = false
			perm1.ID = 0
			err := ps.PostPerm(&perm1)
			g.Assert(err == nil).IsTrue()
			g.Assert(perm1.ID != 0).IsTrue()
			getperm, err := ps.GetPerm(&model.User{ID: 1}, &model.Repo{ID: 2})
			g.Assert(err == nil).IsTrue()
			g.Assert(getperm.Read).IsTrue()
			g.Assert(getperm.Write).IsTrue()
			g.Assert(getperm.Admin).IsFalse()
		})

		g.It("Should Get a Perm", func() {
			ps.PostPerm(&model.Perm{
				UserID: 1,
				RepoID: 2,
				Read:   true,
				Write:  true,
				Admin:  true,
			})
			getperm, err := ps.GetPerm(&model.User{ID: 1}, &model.Repo{ID: 2})
			g.Assert(err == nil).IsTrue()
			g.Assert(getperm.ID != 0).IsTrue()
			g.Assert(getperm.Admin).IsTrue()
			g.Assert(getperm.Write).IsTrue()
			g.Assert(getperm.Admin).IsTrue()
			g.Assert(getperm.UserID).Equal(int64(1))
			g.Assert(getperm.RepoID).Equal(int64(2))
		})

		g.It("Should Del a Perm", func() {
			perm1 := model.Perm{
				UserID: 1,
				RepoID: 2,
				Read:   true,
				Write:  true,
				Admin:  true,
			}
			ps.PostPerm(&perm1)
			_, err1 := ps.GetPerm(&model.User{ID: 1}, &model.Repo{ID: 2})
			err2 := ps.DelPerm(&perm1)
			_, err3 := ps.GetPerm(&model.User{ID: 1}, &model.Repo{ID: 2})
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})
	})
}
