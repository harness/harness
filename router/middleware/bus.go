package middleware

import (
	"github.com/drone/drone/bus"
	"github.com/gin-gonic/gin"
)

func Bus() gin.HandlerFunc {
	bus_ := bus.New()
	return func(c *gin.Context) {
		bus.ToContext(c, bus_)
		c.Next()
	}
}
