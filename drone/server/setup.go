// +build !enterprise

package server

import (
	"github.com/cncd/queue"
	"github.com/drone/drone/model"
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

func setupPubsub(c *cli.Context)          {}
func setupStream(c *cli.Command)          {}
func setupRegistryService(c *cli.Command) {}
func setupSecretService(c *cli.Command)   {}
func setupGatingService(c *cli.Command)   {}
