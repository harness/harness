package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Broker handles connections to the embedded message broker.
func Broker(c *gin.Context) {
	broker := c.MustGet("broker").(http.Handler)
	broker.ServeHTTP(c.Writer, c.Request)
}
