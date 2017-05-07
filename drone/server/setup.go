// +build !enterprise

package server

import (
	"github.com/cncd/queue"
	"github.com/drone/drone/model"
	"github.com/drone/drone/plugins/registry"
	"github.com/drone/drone/plugins/secrets"
	"github.com/drone/drone/store"
	"github.com/drone/drone/store/datastore"

	"github.com/urfave/cli"
)

func setupStore(c *cli.Context) store.Store {
	return datastore.New(
		c.String("driver"),
		c.String("datasource"),
	)
}

func setupQueue(c *cli.Context, s store.Store) queue.Queue {
	return model.WithTaskStore(queue.New(), s)
}

func setupSecretService(c *cli.Context, s store.Store) model.SecretService {
	return secrets.New(s)
}

func setupRegistryService(c *cli.Context, s store.Store) model.RegistryService {
	return registry.New(s)
}

func setupPubsub(c *cli.Context)        {}
func setupStream(c *cli.Command)        {}
func setupGatingService(c *cli.Command) {}
