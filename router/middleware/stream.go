package middleware

import (
	"github.com/drone/drone/stream"

	"github.com/codegangsta/cli"
	"github.com/gin-gonic/gin"
)

// Stream is a middleware function that initializes the Stream and attaches to
// the context of every http.Request.
func Stream(cli *cli.Context) gin.HandlerFunc {
	v := stream.New()
	return func(c *gin.Context) {
		stream.ToContext(c, v)
	}
}
