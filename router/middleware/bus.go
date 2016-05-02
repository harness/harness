package middleware

import (
	"github.com/drone/drone/bus"

	"github.com/codegangsta/cli"
	"github.com/gin-gonic/gin"
)

// Bus is a middleware function that initializes the Event Bus and attaches to
// the context of every http.Request.
func Bus(cli *cli.Context) gin.HandlerFunc {
	v := bus.New()
	return func(c *gin.Context) {
		bus.ToContext(c, v)
	}
}
