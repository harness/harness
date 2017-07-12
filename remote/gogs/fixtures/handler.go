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
	e.GET("/api/v1/repos/:owner/:name", getRepo)
	e.GET("/api/v1/repos/:owner/:name/raw/:commit/:file", getRepoFile)
	e.GET("/api/v1/repos/:owner/:name/hooks", getRepoHooks)
	e.POST("/api/v1/repos/:owner/:name/hooks", createRepoHook)
	e.GET("/api/v1/repos/:owner/:name/hooks/:id", deleteRepoHook)
	e.GET("/api/v1/user/repos", getUserRepos)

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

func getRepoFile(c *gin.Context) {
	if c.Param("file") == "file_not_found" {
		c.String(404, "")
	}
	if c.Param("commit") == "v1.0.0" || c.Param("commit") == "9ecad50" {
		c.String(200, repoFilePayload)
	}
	c.String(404, "")
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

func createRepoHook(c *gin.Context) {
	switch c.Param("name") {
	case "hooks_not_found", "repo_no_hooks":
		c.String(404, "")
	case "hook_empty":
		c.String(200, "{}")
	default:
		c.String(200, createRepoHookPayload)
	}
}

func deleteRepoHook(c *gin.Context) {
	switch c.Param("id") {
	case "hook_not_found":
		c.String(404, "")
	default:
		c.String(204, "")
	}
}

func getUserRepos(c *gin.Context) {
	switch c.Request.Header.Get("Authorization") {
	case "token repos_not_found":
		c.String(404, "")
	default:
		c.String(200, userRepoPayload)
	}
}

const repoFilePayload = `{ platform: linux/amd64 }`

const repoPayload = `
{
  "owner": {
    "username": "test_name",
    "email": "octocat@github.com",
    "avatar_url": "https:\/\/secure.gravatar.com\/avatar\/8c58a0be77ee441bb8f8595b7f1b4e87"
  },
  "full_name": "test_name\/repo_name",
  "private": true,
  "html_url": "http:\/\/localhost\/test_name\/repo_name",
  "clone_url": "http:\/\/localhost\/test_name\/repo_name.git",
  "permissions": {
    "admin": true,
    "push": true,
    "pull": true
  }
}
`

const userRepoPayload = `
[
  {
    "owner": {
      "username": "test_name",
      "email": "octocat@github.com",
      "avatar_url": "https:\/\/secure.gravatar.com\/avatar\/8c58a0be77ee441bb8f8595b7f1b4e87"
    },
    "full_name": "test_name\/repo_name",
    "private": true,
    "html_url": "http:\/\/localhost\/test_name\/repo_name",
    "clone_url": "http:\/\/localhost\/test_name\/repo_name.git",
    "permissions": {
      "admin": true,
      "push": true,
      "pull": true
    }
  }
]
`

const repoHookPayload = `
[
  {
    "id": 14,
    "type": "gogs",
    "events": [
      "create",
      "push"
    ],
    "active": true,
    "config": {
      "content_type": "json",
      "url": "http:\/\/localhost\/test_name\/repo_name\/settings\/hooks\/14"
    },
    "updated_at": "2015-08-29T18:25:52+08:00",
    "created_at": "2015-08-27T20:17:36+08:00"
  }
]
`

const createRepoHookPayload = `
{
  "id": 14,
  "type": "gogs",
  "events": [
    "create",
    "push"
  ],
  "active": true,
  "config": {
    "content_type": "json",
    "url": "http:\/\/localhost\/test_name\/repo_name\/settings\/hooks\/14"
  },
  "updated_at": "2015-08-29T18:25:52+08:00",
  "created_at": "2015-08-27T20:17:36+08:00"
}
`
