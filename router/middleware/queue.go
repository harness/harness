package middleware

import (
	"github.com/drone/drone/queue"

	"github.com/codegangsta/cli"
	"github.com/gin-gonic/gin"
)

// Queue is a middleware function that initializes the Queue and attaches to
// the context of every http.Request.
func Queue(cli *cli.Context) gin.HandlerFunc {
	v := queue.New()
	return func(c *gin.Context) {
		queue.ToContext(c, v)
	}
}
