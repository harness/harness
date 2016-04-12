package middleware

import (
	"time"

	"github.com/drone/drone/cache"

	"github.com/gin-gonic/gin"
	"github.com/ianschenck/envflag"
)

var ttl = envflag.Duration("CACHE_TTL", time.Minute*15, "")

// Cache is a middleware function that initializes the Cache and attaches to
// the context of every http.Request.
func Cache() gin.HandlerFunc {
	cc := cache.NewTTL(*ttl)
	return func(c *gin.Context) {
		cache.ToContext(c, cc)
		c.Next()
	}
}
