package context

import (
	"database/sql"

	"github.com/drone/drone/engine"
	"github.com/drone/drone/remote"
	"github.com/gin-gonic/gin"
)

func SetDatabase(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("database", db)
		c.Next()
	}
}

func Database(c *gin.Context) *sql.DB {
	return c.MustGet("database").(*sql.DB)
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
