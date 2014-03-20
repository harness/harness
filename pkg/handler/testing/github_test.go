package testing

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/handler"
	"github.com/drone/drone/pkg/model"

	dbtest "github.com/drone/drone/pkg/database/testing"
	. "github.com/smartystreets/goconvey/convey"
)

// Tests the ability to create GitHub repositories.
func Test_GitHubCreate(t *testing.T) {
	// seed the database with values
	SetupFixtures()
	defer TeardownFixtures()

	// mock request
	req := http.Request{}
	req.Form = url.Values{}

	// get user that will add repositories
	user, _ := database.GetUser(1)
	settings := database.SettingsMust()

	Convey("Given request to setup github repo", t, func() {

		Convey("When repository is public", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "public")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateGithub(res, &req, user)
			repo, _ := database.GetRepoSlug(settings.GitHubDomain + "/example/public")

			Convey("The repository is created", func() {
				So(err, ShouldBeNil)
				So(repo, ShouldNotBeNil)
				So(repo.ID, ShouldNotEqual, 0)
				So(repo.Owner, ShouldEqual, "example")
				So(repo.Name, ShouldEqual, "public")
				So(repo.Host, ShouldEqual, settings.GitHubDomain)
				So(repo.TeamID, ShouldEqual, 0)
				So(repo.UserID, ShouldEqual, user.ID)
				So(repo.Private, ShouldEqual, false)
				So(repo.SCM, ShouldEqual, "git")
			})
			Convey("The repository is public", func() {
				So(repo.Private, ShouldEqual, false)
			})
		})

		Convey("When repository is private", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "private")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateGithub(res, &req, user)
			repo, _ := database.GetRepoSlug(settings.GitHubDomain + "/example/private")

			Convey("The repository is created", func() {
				So(err, ShouldBeNil)
				So(repo, ShouldNotBeNil)
				So(repo.ID, ShouldNotEqual, 0)
			})
			Convey("The repository is private", func() {
				So(repo.Private, ShouldEqual, true)
			})
		})

		Convey("When repository is not found", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "notfound")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateGithub(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to find GitHub repository example/notfound.")
			})

			Convey("The repository is not created", func() {
				_, err := database.GetRepoSlug("example/notfound")
				So(err, ShouldNotBeNil)
				So(err, ShouldEqual, sql.ErrNoRows)
			})
		})

		Convey("When repository hook is not writable", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "hookerr")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateGithub(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to add Hook to your GitHub repository.")
			})

			Convey("The repository is not created", func() {
				_, err := database.GetRepoSlug("example/hookerr")
				So(err, ShouldNotBeNil)
				So(err, ShouldEqual, sql.ErrNoRows)
			})
		})

		Convey("When repository ssh key is not writable", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "keyerr")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateGithub(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to add Public Key to your GitHub repository.")
			})

			Convey("The repository is not created", func() {
				_, err := database.GetRepoSlug("example/keyerr")
				So(err, ShouldNotBeNil)
				So(err, ShouldEqual, sql.ErrNoRows)
			})
		})

		Convey("When a team is provided", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "team")
			req.Form.Set("team", "drone")
			res := httptest.NewRecorder()

			// invoke handler
			err := handler.RepoCreateGithub(res, &req, user)
			team, _ := database.GetTeamSlug("drone")
			repo, _ := database.GetRepoSlug(settings.GitHubDomain + "/example/team")

			Convey("The repository is created", func() {
				So(err, ShouldBeNil)
				So(repo, ShouldNotBeNil)
				So(repo.ID, ShouldNotEqual, 0)
			})

			Convey("The team should be set", func() {
				So(repo.TeamID, ShouldEqual, team.ID)
			})
		})

		Convey("When a team is not found", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "public")
			req.Form.Set("team", "faketeam")
			res := httptest.NewRecorder()
			err := handler.RepoCreateGithub(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to find Team faketeam.")
			})
		})

		Convey("When a team is forbidden", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "public")
			req.Form.Set("team", "golang")
			res := httptest.NewRecorder()
			err := handler.RepoCreateGithub(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Invalid permission to access Team golang.")
			})
		})
	})
}

// this code should be refactored and centralized, but for now
// it is just proof-of-concepting a testing strategy, so we'll
// revisit later.

// mux is the HTTP request multiplexer used with the test server.
var mux *http.ServeMux

// server is a test HTTP server used to provide mock API responses.
var server *httptest.Server

func SetupFixtures() {
	dbtest.Setup()

	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
	url, _ := url.Parse(server.URL)

	// set database to use a localhost url for GitHub
	settings := model.Settings{}
	settings.GitHubKey = "123"
	settings.GitHubSecret = "abc"
	settings.GitHubApiUrl = url.String() // normall would be "https://api.github.com"
	settings.GitHubDomain = url.Host     // normally would be "github.com"
	settings.Scheme = url.Scheme
	settings.Domain = "localhost"
	database.SaveSettings(&settings)

	// -----------------------------------------------------------------------------------
	// fixture to return a public repository and successfully
	// create a commit hook.

	mux.HandleFunc("/repos/example/public", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "public",
			"full_name": "example/public",
			"private": false
		}`)
	})

	mux.HandleFunc("/repos/example/public/hooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"url": "https://api.github.com/repos/example/public/hooks/1",
			"name": "web",
			"events": [ "push", "pull_request" ],
			"id": 1
		}`)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a private repository and successfully
	// create a commit hook and ssh deploy key

	mux.HandleFunc("/repos/example/private", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "private",
			"full_name": "example/private",
			"private": true
		}`)
	})

	mux.HandleFunc("/repos/example/private/hooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"url": "https://api.github.com/repos/example/private/hooks/1",
			"name": "web",
			"events": [ "push", "pull_request" ],
			"id": 1
		}`)
	})

	mux.HandleFunc("/repos/example/private/keys", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"id": 1,
			"key": "ssh-rsa AAA...",
			"url": "https://api.github.com/user/keys/1",
			"title": "octocat@octomac"
		}`)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a not found when accessing a github repository.

	mux.HandleFunc("/repos/example/notfound", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a public repository and successfully
	// create a commit hook.

	mux.HandleFunc("/repos/example/hookerr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "hookerr",
			"full_name": "example/hookerr",
			"private": false
		}`)
	})

	mux.HandleFunc("/repos/example/hookerr/hooks", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a private repository and successfully
	// create a commit hook and ssh deploy key

	mux.HandleFunc("/repos/example/keyerr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "keyerr",
			"full_name": "example/keyerr",
			"private": true
		}`)
	})

	mux.HandleFunc("/repos/example/keyerr/hooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"url": "https://api.github.com/repos/example/keyerr/hooks/1",
			"name": "web",
			"events": [ "push", "pull_request" ],
			"id": 1
		}`)
	})

	mux.HandleFunc("/repos/example/keyerr/keys", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a public repository and successfully to
	// test adding a team.

	mux.HandleFunc("/repos/example/team", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "team",
			"full_name": "example/team",
			"private": false
		}`)
	})

	mux.HandleFunc("/repos/example/team/hooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"url": "https://api.github.com/repos/example/team/hooks/1",
			"name": "web",
			"events": [ "push", "pull_request" ],
			"id": 1
		}`)
	})
}

func TeardownFixtures() {
	dbtest.Teardown()
	server.Close()
}

/*

// response for querying a repo
var repoGet = `{
  "name": "Hello-World",
  "full_name": "octocat/Hello-World",
  "owner": {
    "login": "octocat"
  },
  "private": false,
  "git_url": "git://github.com/octocat/Hello-World.git",
  "ssh_url": "git@github.com:octocat/Hello-World.git",
  "clone_url": "https://github.com/octocat/Hello-World.git"
}`

// response for querying a private repo
var repoPrivateGet = `{
  "name": "Hello-World",
  "full_name": "octocat/Hello-World",
  "owner": {
    "login": "octocat"
  },
  "private": true,
  "git_url": "git://github.com/octocat/Hello-World.git",
  "ssh_url": "git@github.com:octocat/Hello-World.git",
  "clone_url": "https://github.com/octocat/Hello-World.git"
}`

// response for creating a key
var keyAdd = `
{
  "id": 1,
  "key": "ssh-rsa AAA...",
  "url": "https://api.github.com/user/keys/1",
  "title": "octocat@octomac"
}
`

// response for creating a hook
var hookAdd = `
{
  "url": "https://api.github.com/repos/octocat/Hello-World/hooks/1",
  "updated_at": "2011-09-06T20:39:23Z",
  "created_at": "2011-09-06T17:26:27Z",
  "name": "web",
  "events": [
    "push",
    "pull_request"
  ],
  "active": true,
  "config": {
    "url": "http://example.com",
    "content_type": "json"
  },
  "id": 1
}
`
*/
