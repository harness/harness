package middleware

import (
	"github.com/drone/drone/remote"
	"github.com/gin-gonic/gin"
)

// Remote is a middleware function that initializes the Remote and attaches to
// the context of every http.Request.
func Remote(v remote.Remote) gin.HandlerFunc {
	return func(c *gin.Context) {
		remote.ToContext(c, v)
	}
}
