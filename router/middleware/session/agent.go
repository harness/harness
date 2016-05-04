package session

import (
	"github.com/drone/drone/shared/token"
	"github.com/gin-gonic/gin"
)

// AuthorizeAgent authorizes requsts from build agents to access the queue.
func AuthorizeAgent(c *gin.Context) {
	secret := c.MustGet("agent").(string)
	if secret == "" {
		c.String(401, "invalid or empty token.")
		return
	}

	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return secret, nil
	})
	if err != nil {
		c.String(500, "invalid or empty token. %s", err)
		c.Abort()
	} else if parsed.Kind != token.AgentToken {
		c.String(403, "invalid token. please use an agent token")
		c.Abort()
	} else {
		c.Next()
	}
}
