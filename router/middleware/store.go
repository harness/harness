package middleware

import (
	"context"

	"github.com/cncd/logging"
	"github.com/cncd/pubsub"
	"github.com/cncd/queue"
	"github.com/drone/drone/model"
	"github.com/drone/drone/plugins/registry"
	"github.com/drone/drone/plugins/secrets"
	"github.com/drone/drone/plugins/sender"
	"github.com/drone/drone/server"
	"github.com/drone/drone/store"
	"github.com/drone/drone/store/datastore"
	"github.com/urfave/cli"

	"github.com/gin-gonic/gin"
)

// Store is a middleware function that initializes the Datastore and attaches to
// the context of every http.Request.
func Store(cli *cli.Context) gin.HandlerFunc {
	v := setupStore(cli)

	// HACK during refactor period. Please ignore my mess.

	// storage
	server.Config.Storage.Files = v

	// services
	server.Config.Services.Queue = model.WithTaskStore(queue.New(), v)
	server.Config.Services.Logs = logging.New()
	server.Config.Services.Pubsub = pubsub.New()
	server.Config.Services.Pubsub.Create(context.Background(), "topic/events")
	server.Config.Services.Registries = registry.New(v)
	server.Config.Services.Secrets = secrets.New(v)
	server.Config.Services.Senders = sender.New(v)
	if endpoint := cli.String("registry-service"); endpoint != "" {
		server.Config.Services.Registries = registry.NewRemote(endpoint)
	}
	if endpoint := cli.String("secret-service"); endpoint != "" {
		server.Config.Services.Secrets = secrets.NewRemote(endpoint)
	}
	if endpoint := cli.String("gating-service"); endpoint != "" {
		server.Config.Services.Senders = sender.NewRemote(endpoint)
	}

	// server configuration
	server.Config.Server.Cert = cli.String("server-cert")
	server.Config.Server.Key = cli.String("server-key")
	server.Config.Server.Pass = cli.String("agent-secret")
	server.Config.Server.Host = cli.String("server-host")
	server.Config.Server.Port = cli.String("server-addr")
	server.Config.Pipeline.Networks = cli.StringSlice("network")
	server.Config.Pipeline.Volumes = cli.StringSlice("volumes")
	server.Config.Pipeline.Privileged = cli.StringSlice("escalate")
	// server.Config.Server.Open = cli.Bool("open")
	// server.Config.Server.Orgs = sliceToMap(cli.StringSlice("orgs"))
	// server.Config.Server.Admins = sliceToMap(cli.StringSlice("admin"))

	return func(c *gin.Context) {
		store.ToContext(c, v)
		c.Next()
	}
}

// helper function to create the datastore from the CLI context.
func setupStore(c *cli.Context) store.Store {
	return datastore.New(
		c.String("driver"),
		c.String("datasource"),
	)
}

// helper function to convert a string slice to a map.
func sliceToMap(s []string) map[string]struct{} {
	v := map[string]struct{}{}
	for _, ss := range s {
		v[ss] = struct{}{}
	}
	return v
}
