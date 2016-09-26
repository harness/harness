package middleware

import (
	"sync"

	handlers "github.com/drone/drone/server"

	"github.com/codegangsta/cli"
	"github.com/drone/mq/server"
	"github.com/drone/mq/stomp"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

const (
	serverKey = "broker"
	clientKey = "stomp.client" // mirrored from stomp/context
)

// Broker is a middleware function that initializes the broker
// and adds the broker client to the request context.
func Broker(cli *cli.Context) gin.HandlerFunc {
	secret := cli.String("agent-secret")
	if secret == "" {
		logrus.Fatalf("failed to generate token from DRONE_SECRET")
	}

	broker := server.NewServer(
		server.WithCredentials("x-token", secret),
	)
	client := broker.Client()

	var once sync.Once
	return func(c *gin.Context) {
		c.Set(serverKey, broker)
		c.Set(clientKey, client)
		once.Do(func() {
			// this is some really hacky stuff
			// turns out I need to do some refactoring
			// don't judge!
			// will fix in 0.6 release
			ctx := c.Copy()
			client.Connect(
				stomp.WithCredentials("x-token", secret),
			)
			client.Subscribe("/queue/updates", stomp.HandlerFunc(func(m *stomp.Message) {
				go handlers.HandleUpdate(ctx, m.Copy())
			}))
		})
	}
}
