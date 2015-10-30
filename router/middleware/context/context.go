package context

import (
	"github.com/CiscoCloud/drone/engine"
	"github.com/CiscoCloud/drone/remote"
	"github.com/CiscoCloud/drone/store"
	"github.com/gin-gonic/gin"
)

func SetStore(s store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		store.ToContext(c, s)
		c.Next()
	}
}

func SetRemote(remote remote.Remote) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("remote", remote)
		c.Next()
	}
}

func Remote(c *gin.Context) remote.Remote {
	return c.MustGet("remote").(remote.Remote)
}

func SetEngine(engine engine.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("engine", engine)
		c.Next()
	}
}

func Engine(c *gin.Context) engine.Engine {
	return c.MustGet("engine").(engine.Engine)
}
