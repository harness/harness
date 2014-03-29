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
	"github.com/drone/drone/pkg/queue"
	"github.com/drone/drone/pkg/model"

	dbtest "github.com/drone/drone/pkg/database/testing"
	. "github.com/smartystreets/goconvey/convey"
)

// Tests the ability to create GitHub repositories.
func Test_GitLabCreate(t *testing.T) {
	// seed the database with values
	SetupGitlabFixtures()
	defer TeardownGitlabFixtures()

	q := &queue.Queue{}
	gl := handler.NewGitlabHandler(q)

	// mock request
	req := http.Request{}
	req.Form = url.Values{}

	// get user that will add repositories
	user, _ := database.GetUser(1)
	settings := database.SettingsMust()

	Convey("Given request to setup gitlab repo", t, func() {

		Convey("When repository is public", func() {
			req.Form.Set("owner", "example")
			req.Form.Set("name", "public")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := gl.Create(res, &req, user)
			repo, _ := database.GetRepoSlug(settings.GitlabDomain + "/example/public")

			Convey("The repository is created", func() {
				So(err, ShouldBeNil)
				So(repo, ShouldNotBeNil)
				So(repo.ID, ShouldNotEqual, 0)
				So(repo.Owner, ShouldEqual, "example")
				So(repo.Name, ShouldEqual, "public")
				So(repo.Host, ShouldEqual, settings.GitlabDomain)
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
			err := gl.Create(res, &req, user)
			repo, _ := database.GetRepoSlug(settings.GitlabDomain + "/example/private")

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
			err := gl.Create(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "*Gitlab.buildAndExecRequest failed: 404 Not Found")
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
			err := gl.Create(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to add Hook to your GitLab repository.")
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
			err := gl.Create(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to add Public Key to your GitLab repository.")
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
			err := gl.Create(res, &req, user)
			team, _ := database.GetTeamSlug("drone")
			repo, _ := database.GetRepoSlug(settings.GitlabDomain + "/example/team")

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
			err := gl.Create(res, &req, user)

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
			err := gl.Create(res, &req, user)

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


// server is a test HTTP server used to provide mock API responses.
var glServer *httptest.Server


func SetupGitlabFixtures() {
	dbtest.Setup()

	// test server
	mux := http.NewServeMux()
	glServer = httptest.NewServer(mux)
	url, _ := url.Parse(glServer.URL)

	// set database to use a localhost url for GitHub
	settings := model.Settings{}
	settings.GitlabApiUrl = url.String() // normall would be "https://api.github.com"
	settings.GitlabDomain = url.Host     // normally would be "github.com"
	settings.Scheme = url.Scheme
	settings.Domain = "localhost"
	database.SaveSettings(&settings)

	// -----------------------------------------------------------------------------------
	// fixture to return a public repository and successfully
	// create a commit hook.

	mux.HandleFunc("/api/v3/projects/example%2Fpublic", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "public",
			"path_with_namespace": "example/public",
			"public": true
		}`)
	})

	mux.HandleFunc("/api/v3/projects/example%2Fpublic/hooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"url": "https://example.com/example/public/hooks/1",
			"id": 1
		}`)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a private repository and successfully
	// create a commit hook and ssh deploy key

	mux.HandleFunc("/api/v3/projects/example%2Fprivate", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "private",
			"path_with_namespace": "example/private",
			"public": false
		}`)
	})

	mux.HandleFunc("/api/v3/projects/example%2Fprivate/hooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"url": "https://example.com/example/private/hooks/1",
			"id": 1
		}`)
	})

	mux.HandleFunc("/api/v3/projects/example%2Fprivate/keys", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"id": 1,
			"key": "ssh-rsa AAA...",
			"url": "https://api.github.com/user/keys/1",
			"title": "octocat@octomac"
		}`)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a not found when accessing a github repository.

	mux.HandleFunc("/api/v3/projects/example%2Fnotfound", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a public repository and successfully
	// create a commit hook.

	mux.HandleFunc("/api/v3/projects/example%2Fhookerr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "hookerr",
			"path_with_namespace": "example/hookerr",
			"public": true
		}`)
	})

	mux.HandleFunc("/api/v3/projects/example%2Fhookerr/hooks", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a private repository and successfully
	// create a commit hook and ssh deploy key

	mux.HandleFunc("/api/v3/projects/example%2Fkeyerr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "keyerr",
			"path_with_namespace": "example/keyerr",
			"public": false
		}`)
	})

	mux.HandleFunc("/api/v3/projects/example%2Fkeyerr/hooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"url": "https://api.github.com/api/v3/projects/example/keyerr/hooks/1",
			"id": 1
		}`)
	})

	mux.HandleFunc("/api/v3/projects/example%2Fkeyerr/keys", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	})

	// -----------------------------------------------------------------------------------
	// fixture to return a public repository and successfully to
	// test adding a team.

	mux.HandleFunc("/api/v3/projects/example%2Fteam", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "team",
			"path_with_namespace": "example/team",
			"public": true
		}`)
	})

	mux.HandleFunc("/api/v3/projects/example%2Fteam/hooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"url": "https://api.github.com/api/v3/projects/example/team/hooks/1",
			"id": 1
		}`)
	})
}

func TeardownGitlabFixtures() {
	dbtest.Teardown()
	glServer.Close()
}
