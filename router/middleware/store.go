package middleware

import (
	"github.com/codegangsta/cli"
	"github.com/drone/drone/store"
	"github.com/drone/drone/store/datastore"

	"github.com/gin-gonic/gin"
)

// Store is a middleware function that initializes the Datastore and attaches to
// the context of every http.Request.
func Store(cli *cli.Context) gin.HandlerFunc {
	v := setupStore(cli)
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
