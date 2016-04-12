package middleware

import (
	"github.com/drone/drone/store"
	"github.com/drone/drone/store/datastore"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ianschenck/envflag"
)

var (
	database   = envflag.String("DATABASE_DRIVER", "sqlite3", "")
	datasource = envflag.String("DATABASE_CONFIG", "drone.sqlite", "")
)

// Store is a middleware function that initializes the Datastore and attaches to
// the context of every http.Request.
func Store() gin.HandlerFunc {
	db := datastore.New(*database, *datasource)

	logrus.Infof("using database driver %s", *database)
	logrus.Infof("using database config %s", *datasource)

	return func(c *gin.Context) {
		store.ToContext(c, db)
		c.Next()
	}
}
