package server

import (
	"github.com/drone/drone/common"
	"github.com/gin-gonic/gin"
)

// GET /api/agents/token
func GetAgentToken(c *gin.Context) {
	sess := ToSession(c)
	token := &common.Token{}
	token.Kind = common.TokenAgent
	token.Label = "drone-agent"
	tokenstr, err := sess.GenerateToken(token)
	if err != nil {
		c.Fail(500, err)
	} else {
		c.JSON(200, tokenstr)
	}
}
