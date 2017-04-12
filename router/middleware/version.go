package middleware

import (
	"github.com/drone/drone/version"
	"github.com/gin-gonic/gin"
)

// Version is a middleware function that appends the Drone version information
// to the HTTP response. This is intended for debugging and troubleshooting.
func Version(c *gin.Context) {
	c.Header("X-DRONE-VERSION", version.Version.String())
}
