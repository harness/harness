package cache

import (
	"github.com/drone/drone/cache"
	"github.com/gin-gonic/gin"
)

func Default() gin.HandlerFunc {
	cc := cache.Default()
	return func(c *gin.Context) {
		cache.ToContext(c, cc)
		c.Next()
	}
}
