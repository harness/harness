package web

import "github.com/gin-gonic/gin"

// Slack is handler function that handles Slack slash commands.
func Slack(c *gin.Context) {
	text := c.PostForm("text")
	c.String(200, "received message %s", text)
}
