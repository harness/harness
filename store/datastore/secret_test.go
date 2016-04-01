package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func TestSecrets(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("Secrets", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec(rebind("DELETE FROM secrets"))
		})

		g.It("Should set and get a secret", func() {
			secret := &model.Secret{
				RepoID: 1,
				Name:   "foo",
				Value:  "bar",
				Images: []string{"docker", "gcr"},
				Events: []string{"push", "tag"},
			}
			err := s.SetSecret(secret)
			g.Assert(err == nil).IsTrue()
			g.Assert(secret.ID != 0).IsTrue()

			got, err := s.GetSecret(&model.Repo{ID: 1}, secret.Name)
			g.Assert(err == nil).IsTrue()
			g.Assert(got.Name).Equal(secret.Name)
			g.Assert(got.Value).Equal(secret.Value)
			g.Assert(got.Images).Equal(secret.Images)
			g.Assert(got.Events).Equal(secret.Events)
		})

		g.It("Should update a secret", func() {
			secret := &model.Secret{
				RepoID: 1,
				Name:   "foo",
				Value:  "bar",
			}
			s.SetSecret(secret)
			secret.Value = "baz"
			s.SetSecret(secret)

			got, err := s.GetSecret(&model.Repo{ID: 1}, secret.Name)
			g.Assert(err == nil).IsTrue()
			g.Assert(got.Name).Equal(secret.Name)
			g.Assert(got.Value).Equal(secret.Value)
		})

		g.It("Should list secrets", func() {
			s.SetSecret(&model.Secret{
				RepoID: 1,
				Name:   "foo",
				Value:  "bar",
			})
			s.SetSecret(&model.Secret{
				RepoID: 1,
				Name:   "bar",
				Value:  "baz",
			})
			secrets, err := s.GetSecretList(&model.Repo{ID: 1})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(secrets)).Equal(2)
		})

		g.It("Should delete a secret", func() {
			secret := &model.Secret{
				RepoID: 1,
				Name:   "foo",
				Value:  "bar",
			}
			s.SetSecret(secret)

			_, err := s.GetSecret(&model.Repo{ID: 1}, secret.Name)
			g.Assert(err == nil).IsTrue()

			err = s.DeleteSecret(secret)
			g.Assert(err == nil).IsTrue()

			_, err = s.GetSecret(&model.Repo{ID: 1}, secret.Name)
			g.Assert(err != nil).IsTrue("expect a no rows in result set error")
		})
	})
}
