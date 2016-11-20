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
	e.GET("/api/v3/orgs/:org/memberships/:user", getMembership)
	e.GET("/api/v3/user/memberships/orgs/:org", getMembership)

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

func getMembership(c *gin.Context) {
	switch c.Param("org") {
	case "org_not_found":
		c.String(404, "")
	case "github":
		c.String(200, membershipIsMemberPayload)
	default:
		c.String(200, membershipIsOwnerPayload)
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

var membershipIsOwnerPayload = `
{
  "url": "https://api.github.com/orgs/octocat/memberships/octocat",
  "state": "active",
  "role": "admin",
  "organization_url": "https://api.github.com/orgs/octocat",
  "user": {
    "login": "octocat",
    "id": 5555555,
    "avatar_url": "https://github.com/images/error/octocat_happy.gif",
    "gravatar_id": "",
    "url": "https://api.github.com/users/octocat",
    "html_url": "https://github.com/octocat",
    "followers_url": "https://api.github.com/users/octocat/followers",
    "following_url": "https://api.github.com/users/octocat/following{/other_user}",
    "gists_url": "https://api.github.com/users/octocat/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/octocat/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/octocat/subscriptions",
    "organizations_url": "https://api.github.com/users/octocat/orgs",
    "repos_url": "https://api.github.com/users/octocat/repos",
    "events_url": "https://api.github.com/users/octocat/events{/privacy}",
    "received_events_url": "https://api.github.com/users/octocat/received_events",
    "type": "User",
    "site_admin": false
  },
  "organization": {
    "login": "octocat",
    "id": 5555556,
    "url": "https://api.github.com/orgs/octocat",
    "repos_url": "https://api.github.com/orgs/octocat/repos",
    "events_url": "https://api.github.com/orgs/octocat/events",
    "hooks_url": "https://api.github.com/orgs/octocat/hooks",
    "issues_url": "https://api.github.com/orgs/octocat/issues",
    "members_url": "https://api.github.com/orgs/octocat/members{/member}",
    "public_members_url": "https://api.github.com/orgs/octocat/public_members{/member}",
    "avatar_url": "https://github.com/images/error/octocat_happy.gif",
    "description": ""
  }
}
`

var membershipIsMemberPayload = `
{
  "url": "https://api.github.com/orgs/github/memberships/octocat",
  "state": "active",
  "role": "member",
  "organization_url": "https://api.github.com/orgs/github",
  "user": {
    "login": "octocat",
    "id": 5555555,
    "avatar_url": "https://github.com/images/error/octocat_happy.gif",
    "gravatar_id": "",
    "url": "https://api.github.com/users/octocat",
    "html_url": "https://github.com/octocat",
    "followers_url": "https://api.github.com/users/octocat/followers",
    "following_url": "https://api.github.com/users/octocat/following{/other_user}",
    "gists_url": "https://api.github.com/users/octocat/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/octocat/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/octocat/subscriptions",
    "organizations_url": "https://api.github.com/users/octocat/orgs",
    "repos_url": "https://api.github.com/users/octocat/repos",
    "events_url": "https://api.github.com/users/octocat/events{/privacy}",
    "received_events_url": "https://api.github.com/users/octocat/received_events",
    "type": "User",
    "site_admin": false
  },
  "organization": {
    "login": "octocat",
    "id": 5555557,
    "url": "https://api.github.com/orgs/github",
    "repos_url": "https://api.github.com/orgs/github/repos",
    "events_url": "https://api.github.com/orgs/github/events",
    "hooks_url": "https://api.github.com/orgs/github/hooks",
    "issues_url": "https://api.github.com/orgs/github/issues",
    "members_url": "https://api.github.com/orgs/github/members{/member}",
    "public_members_url": "https://api.github.com/orgs/github/public_members{/member}",
    "avatar_url": "https://github.com/images/error/octocat_happy.gif",
    "description": ""
  }
}
`
