package middleware

import (
	"github.com/drone/drone/store"
	"github.com/urfave/cli"

	"github.com/gin-gonic/gin"
)

// Store is a middleware function that initializes the Datastore and attaches to
// the context of every http.Request.
func Store(cli *cli.Context, v store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		store.ToContext(c, v)
		c.Next()
	}
}
