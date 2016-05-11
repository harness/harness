package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func TestAgents(t *testing.T) {
	db := openTest()
	defer db.Close()
	s := From(db)

	g := goblin.Goblin(t)
	g.Describe("Agents", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM agents")
		})

		g.It("Should update", func() {
			agent := model.Agent{
				Address:  "127.0.0.1",
				Platform: "linux/amd64",
			}
			err1 := s.CreateAgent(&agent)
			agent.Platform = "windows/amd64"
			err2 := s.UpdateAgent(&agent)

			getagent, err3 := s.GetAgent(agent.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(agent.ID).Equal(getagent.ID)
			g.Assert(agent.Platform).Equal(getagent.Platform)
		})

		g.It("Should create", func() {
			agent := model.Agent{
				Address:  "127.0.0.1",
				Platform: "linux/amd64",
			}
			err := s.CreateAgent(&agent)
			g.Assert(err == nil).IsTrue()
			g.Assert(agent.ID != 0).IsTrue()
		})

		g.It("Should get by ID", func() {
			agent := model.Agent{
				Address:  "127.0.0.1",
				Platform: "linux/amd64",
			}

			s.CreateAgent(&agent)
			getagent, err := s.GetAgent(agent.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(agent.ID).Equal(getagent.ID)
			g.Assert(agent.Address).Equal(getagent.Address)
			g.Assert(agent.Platform).Equal(getagent.Platform)
		})

		g.It("Should get by IP address", func() {
			agent := model.Agent{
				Address:  "127.0.0.1",
				Platform: "linux/amd64",
			}
			s.CreateAgent(&agent)
			getagent, err := s.GetAgentAddr(agent.Address)
			g.Assert(err == nil).IsTrue()
			g.Assert(agent.ID).Equal(getagent.ID)
			g.Assert(agent.Address).Equal(getagent.Address)
			g.Assert(agent.Platform).Equal(getagent.Platform)
		})

		g.It("Should enforce unique IP address", func() {
			agent1 := model.Agent{
				Address:  "127.0.0.1",
				Platform: "linux/amd64",
			}
			agent2 := model.Agent{
				Address:  "127.0.0.1",
				Platform: "linux/amd64",
			}
			err1 := s.CreateAgent(&agent1)
			err2 := s.CreateAgent(&agent2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should list", func() {
			agent1 := model.Agent{
				Address:  "127.0.0.1",
				Platform: "linux/amd64",
			}
			agent2 := model.Agent{
				Address:  "localhost",
				Platform: "linux/amd64",
			}
			s.CreateAgent(&agent1)
			s.CreateAgent(&agent2)
			agents, err := s.GetAgentList()
			g.Assert(err == nil).IsTrue()
			g.Assert(len(agents)).Equal(2)
			g.Assert(agents[0].Address).Equal(agent1.Address)
			g.Assert(agents[0].Platform).Equal(agent1.Platform)
		})

		// g.It("Should delete", func() {
		// 	user := model.User{
		// 		Login: "joe",
		// 		Email: "foo@bar.com",
		// 		Token: "e42080dddf012c718e476da161d21ad5",
		// 	}
		// 	s.CreateUser(&user)
		// 	_, err1 := s.GetUser(user.ID)
		// 	err2 := s.DeleteUser(&user)
		// 	_, err3 := s.GetUser(user.ID)
		// 	g.Assert(err1 == nil).IsTrue()
		// 	g.Assert(err2 == nil).IsTrue()
		// 	g.Assert(err3 == nil).IsFalse()
		// })
	})
}
