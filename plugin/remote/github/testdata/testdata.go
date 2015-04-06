package testdata

import (
	"net/http"
	"net/http/httptest"
)

// setup a mock server for testing purposes.
func NewServer() *httptest.Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// handle requests and serve mock data
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// evaluate the path to serve a dummy data file
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Write(accessTokenPayload)
			return
		case "/user":
			w.Write(userPayload)
			return
		case "/user/emails":
			w.Write(userEmailsPayload)
			return
		case "/user/repos":
			w.Write(userReposPayload)
			return
		case "/user/orgs":
			w.Write(userOrgsPayload)
			return
		case "/orgs/octocats-inc/repos":
			w.Write(userReposPayload)
			return
		case "/repos/octocat/Hello-World/contents/.drone.yml":
			w.Write(droneYamlPayload)
			return
		case "/repos/octocat/Hello-World/hooks":
			switch r.Method {
			case "POST":
				w.Write(createHookPayload)
				return
			}
		case "/repos/octocat/Hola-Mundo/hooks":
			switch r.Method {
			case "POST":
				w.Write(createHookPayload)
				return
			}
		case "/repos/octocat/Hola-Mundo/keys":
			switch r.Method {
			case "POST":
				w.Write(createKeyPayload)
				return
			}
		}

		// else return a 404
		http.NotFound(w, r)
	})

	// return the server to the client which
	// will need to know the base URL path
	return server
}

var accessTokenPayload = []byte(`access_token=sekret&scope=repo%2Cuser%3Aemail&token_type=bearer`)

var userPayload = []byte(`
{
  "login": "octocat",
  "id": 1,
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
  "site_admin": false,
  "name": "monalisa octocat",
  "company": "GitHub",
  "blog": "https://github.com/blog",
  "location": "San Francisco",
  "email": "octocat@github.com",
  "hireable": false,
  "bio": "There once was...",
  "public_repos": 2,
  "public_gists": 1,
  "followers": 20,
  "following": 0,
  "created_at": "2008-01-14T04:33:35Z",
  "updated_at": "2008-01-14T04:33:35Z",
  "total_private_repos": 100,
  "owned_private_repos": 100,
  "private_gists": 81,
  "disk_usage": 10000,
  "collaborators": 8,
  "plan": {
    "name": "Medium",
    "space": 400,
    "private_repos": 20,
    "collaborators": 0
  }
}
`)

var userEmailsPayload = []byte(`
[
  {
    "email": "octocat@github.com",
    "verified": true,
    "primary": true
  }
]
`)

// sample repository list
var userReposPayload = []byte(`
[
	{
		"owner": {
			"login": "octocat",
			"id":    1
		},
		"id":        1296269,
		"name":      "Hello-World",
		"full_name": "octocat/Hello-World",
		"private":   true,
		"url":       "https://api.github.com/repos/octocat/Hello-World",
		"html_url":  "https://github.com/octocat/Hello-World",
		"clone_url": "https://github.com/octocat/Hello-World.git",
		"git_url":   "git://github.com/octocat/Hello-World.git",
		"ssh_url":   "git@github.com:octocat/Hello-World.git",
		"permissions": {
			"admin": true,
			"push":  true,
			"pull":  true
		}
	},
	{
		"owner": {
			"login": "octocat",
			"id":    1
		},
		"id":        9626921,
		"name":      "Hola-Mundo",
		"full_name": "octocat/Hola-Mundo",
		"private":   false,
		"url":       "https://api.github.com/repos/octocat/Hola-Mundo",
		"html_url":  "https://github.com/octocat/Hola-Mundo",
		"clone_url": "https://github.com/octocat/Hola-Mundo.git",
		"git_url":   "git://github.com/octocat/Hola-Mundo.git",
		"ssh_url":   "git@github.com:octocat/Hola-Mundo.git",
		"permissions": {
			"admin": false,
			"push":  false,
			"pull":  true
		}
	}
]
`)

var emptySetPayload = []byte(`[]`)
var emptyObjPayload = []byte(`{}`)

// sample org list response
var userOrgsPayload = []byte(`
[
	{ "login": "octocats-inc", "id": 1 }
]
`)

// sample content response for .drone.yml request
var droneYamlPayload = []byte(`
{
	"type":     "file",
	"encoding": "base64",
	"name":     ".drone.yml",
	"path":     ".drone.yml",
	"content":  "aW1hZ2U6IGdv"
}
`)

// sample create hook response
var createHookPayload = []byte(`
{
	"id":     1,
	"name":   "web",
	"events": [ "push", "pull_request" ],
	"active": true,
	"config": {
		"url": "http://example.com",
		"content_type": "json"
	}
}
`)

// sample create hook response
var createKeyPayload = []byte(`
{
	"id": 1,
	"key": "ssh-rsa AAA...",
	"url": "https://api.github.com/user/keys/1",
	"title": "octocat@octomac"
}
`)
