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
	e.POST("/site/oauth2/access_token", getOauth)
	e.GET("/2.0/repositories/:owner/:name", getRepo)
	e.GET("/2.0/repositories/:owner/:name/hooks", getRepoHooks)
	e.GET("/1.0/repositories/:owner/:name/src/:commit/:file", getRepoFile)
	e.DELETE("/2.0/repositories/:owner/:name/hooks/:hook", deleteRepoHook)
	e.POST("/2.0/repositories/:owner/:name/hooks", createRepoHook)
	e.POST("/2.0/repositories/:owner/:name/commit/:commit/statuses/build", createRepoStatus)
	e.GET("/2.0/repositories/:owner", getUserRepos)
	e.GET("/2.0/teams/", getUserTeams)
	e.GET("/2.0/user/", getUser)

	return e
}

func getOauth(c *gin.Context) {
	switch c.PostForm("error") {
	case "invalid_scope":
		c.String(500, "")
	}

	switch c.PostForm("code") {
	case "code_bad_request":
		c.String(500, "")
		return
	case "code_user_not_found":
		c.String(200, tokenNotFoundPayload)
		return
	}
	switch c.PostForm("refresh_token") {
	case "refresh_token_not_found":
		c.String(404, "")
	case "refresh_token_is_empty":
		c.Header("Content-Type", "application/json")
		c.String(200, "{}")
	default:
		c.Header("Content-Type", "application/json")
		c.String(200, tokenPayload)
	}
}

func getRepo(c *gin.Context) {
	switch c.Param("name") {
	case "not_found", "repo_unknown", "repo_not_found":
		c.String(404, "")
	default:
		c.String(200, repoPayload)
	}
}

func getRepoHooks(c *gin.Context) {
	switch c.Param("name") {
	case "hooks_not_found", "repo_no_hooks":
		c.String(404, "")
	case "hook_empty":
		c.String(200, "{}")
	default:
		c.String(200, repoHookPayload)
	}
}

func getRepoFile(c *gin.Context) {
	switch c.Param("file") {
	case "file_not_found":
		c.String(404, "")
	default:
		c.String(200, repoFilePayload)
	}
}

func createRepoStatus(c *gin.Context) {
	switch c.Param("name") {
	case "repo_not_found":
		c.String(404, "")
	default:
		c.String(200, "")
	}
}

func createRepoHook(c *gin.Context) {
	c.String(200, "")
}

func deleteRepoHook(c *gin.Context) {
	switch c.Param("name") {
	case "hook_not_found":
		c.String(404, "")
	default:
		c.String(200, "")
	}
}

func getUser(c *gin.Context) {
	switch c.Request.Header.Get("Authorization") {
	case "Bearer user_not_found", "Bearer a87ff679":
		c.String(404, "")
	default:
		c.String(200, userPayload)
	}
}

func getUserTeams(c *gin.Context) {
	switch c.Request.Header.Get("Authorization") {
	case "Bearer teams_not_found", "Bearer c81e728d":
		c.String(404, "")
	default:
		c.String(200, userTeamPayload)
	}
}

func getUserRepos(c *gin.Context) {
	switch c.Request.Header.Get("Authorization") {
	case "Bearer repos_not_found", "Bearer 70efdf2e":
		c.String(404, "")
	default:
		c.String(200, userRepoPayload)
	}
}

const tokenPayload = `
{
	"access_token":"2YotnFZFEjr1zCsicMWpAA",
	"refresh_token":"tGzv3JOkF0XG5Qx2TlKWIA",
	"token_type":"Bearer",
	"expires_in":3600
}
`

const tokenNotFoundPayload = `
{
	"access_token":"user_not_found",
	"refresh_token":"user_not_found",
	"token_type":"Bearer",
	"expires_in":3600
}
`

const repoPayload = `
{
  "full_name": "test_name/repo_name",
  "scm": "git",
  "is_private": true
}
`

const repoHookPayload = `
{
  "pagelen": 10,
  "values": [
  	{
  	  "uuid": "{afe61e14-2c5f-49e8-8b68-ad1fb55fc052}",
  	  "url": "http://127.0.0.1"
  	}
  ],
  "page": 1,
  "size": 1
}
`

const repoFilePayload = `
{
  "data": "{ platform: linux/amd64 }"
}
`

const userPayload = `
{
  "username": "superman",
  "links": {
    "avatar": {
      "href": "http:\/\/i.imgur.com\/ZygP55A.jpg"
    }
  },
  "type": "user"
}
`

const userRepoPayload = `
{
  "page": 1,
  "pagelen": 10,
  "size": 1,
  "values": [
    {
      "links": {
        "avatar": {
            "href": "http:\/\/i.imgur.com\/ZygP55A.jpg"
        }
      },
      "full_name": "test_name/repo_name",
      "scm": "git",
      "is_private": true
    }
  ]
}
`

const userTeamPayload = `
{
  "pagelen": 100,
  "values": [
    {
      "username": "superfriends",
      "links": {
        "avatar": {
          "href": "http:\/\/i.imgur.com\/ZygP55A.jpg"
        }
      },
      "type": "team"
    }
  ]
}
`
