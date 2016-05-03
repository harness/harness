package fixtures

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler returns an http.Handler that is capable of handling a variety of mock
// Bitbucket requests and returning mock responses.
func Handler() http.Handler {
	gin.SetMode(gin.TestMode)

	e := gin.New()
	e.GET("/api/v3/repos/:owner/:name", getRepo)

	return e
}

func getRepo(c *gin.Context) {
	switch c.Param("name") {
	case "repo_not_found":
		c.String(404, "")
	default:
		c.String(200, repoPayload)
	}
}

var repoPayload = `
{
  "owner": {
    "login": "octocat",
    "avatar_url": "https://github.com/images/error/octocat_happy.gif"
  },
  "name": "Hello-World",
  "full_name": "octocat/Hello-World",
  "private": true,
  "html_url": "https://github.com/octocat/Hello-World",
  "clone_url": "https://github.com/octocat/Hello-World.git",
  "language": null,
  "permissions": {
    "admin": true,
    "push": true,
    "pull": true
  }
}
`
