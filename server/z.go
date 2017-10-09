package server

import (
	"github.com/drone/drone/store"
	"github.com/drone/drone/version"
	"github.com/gin-gonic/gin"
)

// Health endpoint returns a 500 if the server state is unhealthy.
func Health(c *gin.Context) {
	if err := store.FromContext(c).Ping(); err != nil {
		c.String(500, err.Error())
		return
	}
	c.String(200, "")
}

// Version endpoint returns the server version and build information.
func Version(c *gin.Context) {
	c.JSON(200, gin.H{
		"source":  "https://github.com/drone/drone",
		"version": version.Version.String(),
	})
}
