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
			db.Exec(rebind("DELETE FROM team_secrets"))
		})

		g.It("Should list all secrets", func() {
			teamSec := &model.TeamSecret{
				Key:   "octocat",
				Name:  "foo",
				Value: "team",
			}

			repoSec := &model.RepoSecret{
				RepoID: 1,
				Name:   "foo",
				Value:  "repo",
			}

			s.SetSecret(repoSec)
			s.SetTeamSecret(teamSec)

			secrets, err := s.GetMergedSecretList(&model.Repo{ID: 1, Owner: "octocat"})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(secrets)).Equal(2)
		})
	})
}
