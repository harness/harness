package middleware

import (
	"github.com/codegangsta/cli"
	"github.com/drone/drone/shared/token"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

const agentKey = "agent"

// Agents is a middleware function that initializes the authorization middleware
// for agents to connect to the queue.
func Agents(cli *cli.Context) gin.HandlerFunc {
	secret := cli.String("agent-secret")
	if secret == "" {
		logrus.Fatalf("failed to generate token from DRONE_AGENT_SECRET")
	}

	t := token.New(token.AgentToken, secret)
	s, err := t.Sign(secret)
	if err != nil {
		logrus.Fatalf("failed to generate token from DRONE_AGENT_SECRET. %s", err)
	}

	logrus.Infof("using agent secret %s", secret)
	logrus.Warnf("agents can connect with token %s", s)

	return func(c *gin.Context) {
		c.Set(agentKey, secret)
	}
}
