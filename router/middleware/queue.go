package middleware

import (
	"github.com/drone/drone/queue"
	"github.com/gin-gonic/gin"
)

func Queue() gin.HandlerFunc {
	queue_ := queue.New()
	return func(c *gin.Context) {
		queue.ToContext(c, queue_)
		c.Next()
	}
}
