package server

import (
	"strconv"

	"github.com/drone/drone/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	common "github.com/drone/drone/pkg/types"
)

// GetAgents accepts a request to retrieve all build
// agents from the datastore and return encoded in JSON
// format.
//
//     GET /api/agents
//
func GetAgents(c *gin.Context) {
	store := ToDatastore(c)
	agents, err := store.AgentList()
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, agents)
	}
}

// PostAgent accepts a request to register a new build
// agent with the system. The registered agent is returned
// from the datastore and return encoded in JSON format.
//
//     POST /api/agents
//
func PostAgent(c *gin.Context) {
	store := ToDatastore(c)

	in := &common.Agent{}
	if !c.BindWith(in, binding.JSON) {
		return
	}

	// attept to fetch the agent from the
	// datastore. If the agent already exists we
	// should re-activate
	agent, err := store.AgentAddr(in.Addr)
	if err != nil {
		agent = &common.Agent{}
		agent.Addr = in.Addr
		agent.Token = types.GenerateToken()
		agent.Active = true
		agent.IsHealthy = true
		err = store.AddAgent(agent)
		if err != nil {
			c.Fail(400, err)
		} else {
			c.JSON(200, agent)
		}
		return
	}

	agent.Active = true
	err = store.SetAgent(agent)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, agent)
	}
}

// DeleteAgent accepts a request to delete a build agent
// from the system.
//
//     DELETE /api/agents/:id
//
func DeleteAgent(c *gin.Context) {
	store := ToDatastore(c)
	idstr := c.Params.ByName("id")
	id, _ := strconv.Atoi(idstr)

	agent, err := store.Agent(int64(id))
	if err != nil {
		c.Fail(404, err)
		return
	}
	agent.Active = false
	err = store.SetAgent(agent)
	if err != nil {
		c.Fail(400, err)
		return
	}

	c.Writer.WriteHeader(200)
}
