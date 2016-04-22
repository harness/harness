package middleware

import (
	"github.com/drone/drone/shared/token"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ianschenck/envflag"
)

var (
	secret = envflag.String("AGENT_SECRET", "", "")
	noauth = envflag.Bool("AGENT_NO_AUTH", false, "")
)

// Agent is a middleware function that initializes the authorization middleware
// for agents to connect to the queue.
func AgentMust() gin.HandlerFunc {

	if *secret == "" {
		logrus.Fatalf("please provide the agent secret to authenticate agent requests")
	}

	t := token.New(token.AgentToken, "")
	s, err := t.Sign(*secret)
	if err != nil {
		logrus.Fatalf("invalid agent secret. %s", err)
	}

	logrus.Infof("using agent secret %s", *secret)
	logrus.Warnf("agents can connect with token %s", s)

	return func(c *gin.Context) {
		parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
			return *secret, nil
		})
		if err != nil {
			c.AbortWithError(403, err)
		} else if parsed.Kind != token.AgentToken {
			c.AbortWithStatus(403)
		} else {
			c.Next()
		}
	}
}
