package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func TestTeamSecrets(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("TeamSecrets", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec(rebind("DELETE FROM team_secrets"))
		})

		g.It("Should set and get a secret", func() {
			secret := &model.TeamSecret{
				Key:    "octocat",
				Name:   "foo",
				Value:  "bar",
				Images: []string{"docker", "gcr"},
				Events: []string{"push", "tag"},
			}
			err := s.SetTeamSecret(secret)
			g.Assert(err == nil).IsTrue()
			g.Assert(secret.ID != 0).IsTrue()

			got, err := s.GetTeamSecret("octocat", secret.Name)
			g.Assert(err == nil).IsTrue()
			g.Assert(got.Name).Equal(secret.Name)
			g.Assert(got.Value).Equal(secret.Value)
			g.Assert(got.Images).Equal(secret.Images)
			g.Assert(got.Events).Equal(secret.Events)
		})

		g.It("Should update a secret", func() {
			secret := &model.TeamSecret{
				Key:   "octocat",
				Name:  "foo",
				Value: "bar",
			}
			s.SetTeamSecret(secret)
			secret.Value = "baz"
			s.SetTeamSecret(secret)

			got, err := s.GetTeamSecret("octocat", secret.Name)
			g.Assert(err == nil).IsTrue()
			g.Assert(got.Name).Equal(secret.Name)
			g.Assert(got.Value).Equal(secret.Value)
		})

		g.It("Should list secrets", func() {
			s.SetTeamSecret(&model.TeamSecret{
				Key:   "octocat",
				Name:  "foo",
				Value: "bar",
			})
			s.SetTeamSecret(&model.TeamSecret{
				Key:   "octocat",
				Name:  "bar",
				Value: "baz",
			})
			secrets, err := s.GetTeamSecretList("octocat")
			g.Assert(err == nil).IsTrue()
			g.Assert(len(secrets)).Equal(2)
		})

		g.It("Should delete a secret", func() {
			secret := &model.TeamSecret{
				Key:   "octocat",
				Name:  "foo",
				Value: "bar",
			}
			s.SetTeamSecret(secret)

			_, err := s.GetTeamSecret("octocat", secret.Name)
			g.Assert(err == nil).IsTrue()

			err = s.DeleteTeamSecret(secret)
			g.Assert(err == nil).IsTrue()

			_, err = s.GetTeamSecret("octocat", secret.Name)
			g.Assert(err != nil).IsTrue("expect a no rows in result set error")
		})
	})
}
