// +build !enterprise

package server

import (
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

func setupQueue(c *cli.Context)           {}
func setupPubsub(c *cli.Context)          {}
func setupStream(c *cli.Command)          {}
func setupRegistryService(c *cli.Command) {}
func setupSecretService(c *cli.Command)   {}
func setupGatingService(c *cli.Command)   {}
