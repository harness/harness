package header

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func SetHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Writer.Header().Add("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Add("X-Frame-Options", "DENY")
		c.Writer.Header().Add("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Add("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Add("Cache-Control", "no-cache")
		c.Writer.Header().Add("Cache-Control", "no-store")
		c.Writer.Header().Add("Cache-Control", "max-age=0")
		c.Writer.Header().Add("Cache-Control", "must-revalidate")
		c.Writer.Header().Add("Cache-Control", "value")
		c.Writer.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		c.Writer.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		//c.Writer.Header().Set("Content-Security-Policy", "script-src 'self' https://cdnjs.cloudflare.com")
		if c.Request.TLS != nil {
			c.Writer.Header().Add("Strict-Transport-Security", "max-age=31536000")
		}

		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization")
			c.Writer.Header().Set("Allow", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.WriteHeader(200)
			return
		}

		c.Next()
	}
}
