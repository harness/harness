package server

import (
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/token"
)

// ShowIndex serves the main Drone application page.
func ShowIndex(c *gin.Context) {
	user := session.User(c)

	var csrf string
	if user != nil {
		csrf, _ = token.New(
			token.CsrfToken,
			user.Login,
		).Sign(user.Hash)
	}

	c.HTML(200, "index.html", gin.H{
		"user": user,
		"csrf": csrf,
	})
}

// ShowLogin is a legacy endpoint that now redirects to
// initiliaze the oauth flow
func ShowLogin(c *gin.Context) {
	if err := c.Query("error"); err != "" {
		c.HTML(500, "error.html", gin.H{"error": err})
		return
	}
	c.Redirect(303, "/authorize")
}

// ShowLoginForm displays a login form for systems like Gogs that do not
// yet support oauth workflows.
func ShowLoginForm(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{})
}
