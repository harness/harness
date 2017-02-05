package middleware

import (
	"os"
	"sync"

	handlers "github.com/drone/drone/server"

	"github.com/codegangsta/cli"
	"github.com/drone/mq/logger"
	"github.com/drone/mq/server"
	"github.com/drone/mq/stomp"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/redlog"
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
		logrus.Fatalf("fatal error. please provide the DRONE_SECRET")
	}

	// setup broker logging.
	log := redlog.New(os.Stderr)
	log.SetLevel(2)
	logger.SetLogger(log)
	if cli.Bool("broker-debug") {
		log.SetLevel(1)
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
