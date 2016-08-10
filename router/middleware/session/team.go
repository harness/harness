package session

import (
	"github.com/gin-gonic/gin"
)

func MustTeamAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := User(c)
		switch {
		case user == nil:
			c.String(401, "User not authorized")
			c.Abort()
		case user.Admin == false:
			c.String(413, "User not authorized")
			c.Abort()
		default:
			c.Next()
		}
	}
}
