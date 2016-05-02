package session

import (
	"github.com/drone/drone/shared/token"
	"github.com/gin-gonic/gin"
)

// AuthorizeAgent authorizes requsts from build agents to access the queue.
func AuthorizeAgent(c *gin.Context) {
	secret := c.MustGet("agent").(string)

	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return secret, nil
	})
	if err != nil {
		c.AbortWithError(403, err)
	} else if parsed.Kind != token.AgentToken {
		c.AbortWithStatus(403)
	} else {
		c.Next()
	}
}
