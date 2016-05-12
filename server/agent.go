package server

import (
	"github.com/drone/drone/store"
	"github.com/gin-gonic/gin"
)

func GetAgents(c *gin.Context) {
	agents, err := store.GetAgentList(c)
	if err != nil {
		c.String(500, "Error getting agent list. %s", err)
		return
	}
	c.JSON(200, agents)
}
