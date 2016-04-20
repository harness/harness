package middleware

import (
	"github.com/drone/drone/stream"
	"github.com/gin-gonic/gin"
)

func Stream() gin.HandlerFunc {
	stream_ := stream.New()
	return func(c *gin.Context) {
		stream.ToContext(c, stream_)
		c.Next()
	}
}
