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

// Test the ability to create Stash repositories
func Test_StashCreate(t *testing.T) {
	// seed the database with values
	SetupStashFixtures()
	defer TeardownStashFixtures()

	// mock request
	req := http.Request{}
	req.Form = url.Values{}

	// get user that will add repositories
	user, _ := database.GetUser(1)
	//settings := database.SettingsMust()

	Convey("Given request to setup stash repo", t, func() {

		Convey("When repository is public", func() {
			req.Form.Set("project", "example")
			req.Form.Set("name", "public")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateStash(res, &req, user)
			repo, _ := database.GetRepoSlug("stash/example/public")

			Convey("The repository is created", func() {
				So(err, ShouldBeNil)
				So(repo, ShouldNotBeNil)
				So(repo.ID, ShouldNotEqual, 0)
				So(repo.Owner, ShouldEqual, "example")
				So(repo.Name, ShouldEqual, "public")
				So(repo.Host, ShouldEqual, "stash")
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
			req.Form.Set("project", "example")
			req.Form.Set("name", "private")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateStash(res, &req, user)
			repo, _ := database.GetRepoSlug("stash/example/private")

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
			req.Form.Set("project", "example")
			req.Form.Set("name", "notfound")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateStash(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to find Stash repository projects/example/repos/notfound.")
			})

			Convey("The repository is not created", func() {
				_, err := database.GetRepoSlug("stash/example/notfound")
				So(err, ShouldNotBeNil)
				So(err, ShouldEqual, sql.ErrNoRows)
			})
		})

		Convey("When repository hook is not writable", func() {
			req.Form.Set("project", "example")
			req.Form.Set("name", "hookerr")
			req.Form.Set("team", "")
			res := httptest.NewRecorder()
			err := handler.RepoCreateStash(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to add Hook to your Stash repository.")
			})

			Convey("The repository is not created", func() {
				_, err := database.GetRepoSlug("stash/example/hookerr")
				So(err, ShouldNotBeNil)
				So(err, ShouldEqual, sql.ErrNoRows)
			})
		})

		// Convey("When repository ssh key is not writable", func() {
		// 	req.Form.Set("project", "example")
		// 	req.Form.Set("name", "keyerr")
		// 	req.Form.Set("team", "")
		// 	res := httptest.NewRecorder()
		// 	err := handler.RepoCreateStash(res, &req, user)

		// 	Convey("The result is an error", func() {
		// 		So(err, ShouldNotBeNil)
		// 		So(err.Error(), ShouldEqual, "Unable to add Public Key to your Stash repository.")
		// 	})

		// 	Convey("The repository is not created", func() {
		// 		_, err := database.GetRepoSlug("stash/example/keyerr")
		// 		So(err, ShouldNotBeNil)
		// 		So(err, ShouldEqual, sql.ErrNoRows)
		// 	})
		// })

		Convey("When a team is provided", func() {
			req.Form.Set("project", "example")
			req.Form.Set("name", "team")
			req.Form.Set("team", "drone")
			res := httptest.NewRecorder()

			// invoke handler
			err := handler.RepoCreateStash(res, &req, user)
			team, _ := database.GetTeamSlug("drone")
			repo, _ := database.GetRepoSlug("stash/example/team")

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
			req.Form.Set("project", "example")
			req.Form.Set("name", "public")
			req.Form.Set("team", "faketeam")
			res := httptest.NewRecorder()
			err := handler.RepoCreateStash(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Unable to find Team faketeam.")
			})
		})

		Convey("When a team is forbidden", func() {
			req.Form.Set("project", "example")
			req.Form.Set("name", "public")
			req.Form.Set("team", "golang")
			res := httptest.NewRecorder()
			err := handler.RepoCreateStash(res, &req, user)

			Convey("The result is an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Invalid permission to access Team golang.")
			})
		})
	})
}

// server is a test HTTP server used to provide mock API responses.
var stashServer *httptest.Server

func SetupStashFixtures() {
	fmt.Println("Stash Testing...")

	dbtest.Setup()

	// test server
	mux := http.NewServeMux()
	stashServer = httptest.NewServer(mux)
	url, _ := url.Parse(stashServer.URL)

	// set database to use a localhost url for Stash
	settings := model.Settings{}
	settings.StashKey = "123"
	settings.StashSecret = "abc"
	settings.StashDomain = url.String()
	settings.StashSshPort = "7999"
	settings.StashHookKey = "hkey"
	settings.StashPrivateKey = "id_stash_test.pem"
	settings.Scheme = url.Scheme
	settings.Domain = "localhost"
	database.SaveSettings(&settings)

	// -----------------------------------------------------------------------
	// fixture to return a public repository and successfully
	// create a commit hook.
	mux.HandleFunc("/rest/api/1.0/projects/example/repos/public", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "public",
			"slug": "example/public",
			"public": true
		}`)
	})

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/public/settings/hooks/hkey/enabled",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
    			"enabled": false,
	    		"details": {
		    	    "key": "key",
			        "name": "key name",
                    "type": "key type",
                    "description": "description",
                    "version": "1.0",
                    "configFormKey": "config form key"
                }
    		}`)
		},
	)

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/public/settings/hooks/hkey/settings",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
    			"enabled": false,
	    		"details": {
		    	    "key": "key",
			        "name": "key name",
                    "type": "key type",
                    "description": "description",
                    "version": "1.0",
                    "configFormKey": "config form key"
                }
    		}`)
		},
	)

	// -----------------------------------------------------------------------
	// fixture to return a private repository and successfully
	// create a commit hook.
	mux.HandleFunc("/rest/api/1.0/projects/example/repos/private", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "public",
			"slug": "example/public",
			"public": false
		}`)
	})

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/private/settings/hooks/hkey/enabled",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
    			"enabled": false,
	    		"details": {
		    	    "key": "key",
			        "name": "key name",
                    "type": "key type",
                    "description": "description",
                    "version": "1.0",
                    "configFormKey": "config form key"
                }
    		}`)
		},
	)

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/private/settings/hooks/hkey/settings",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
    			"enabled": false,
	    		"details": {
		    	    "key": "key",
			        "name": "key name",
                    "type": "key type",
                    "description": "description",
                    "version": "1.0",
                    "configFormKey": "config form key"
                }
    		}`)
		},
	)

	// -----------------------------------------------------------------------
	// fixture to return a not found when accessing a stash repository.
	mux.HandleFunc("/rest/api/1.0/projects/example/repos/notfound", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	// -----------------------------------------------------------------------
	// fixture to return a private repository and then fail to
	// create a commit hook.
	mux.HandleFunc("/rest/api/1.0/projects/example/repos/hookerr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "hookerr",
			"slug": "example/hookerr",
			"public": false
		}`)
	})

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/hookerr/settings/hooks/hkey/enabled",
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Forbidden", http.StatusForbidden)
		},
	)

	// -----------------------------------------------------------------------------------
	// fixture to return a private repository and successfully
	// create a commit hook and then fail to create ssh deploy key
	mux.HandleFunc("/rest/api/1.0/projects/example/repos/keyerr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "keyerr",
			"slug": "example/keyerr",
			"public": false
		}`)
	})

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/keyerr/settings/hooks/hkey/enabled",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
    			"enabled": false,
	    		"details": {
		    	    "key": "key",
			        "name": "key name",
                    "type": "key type",
                    "description": "description",
                    "version": "1.0",
                    "configFormKey": "config form key"
                }
    		}`)
		},
	)

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/keyerr/settings/hooks/hkey/settings",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
    			"enabled": false,
	    		"details": {
		    	    "key": "key",
			        "name": "key name",
                    "type": "key type",
                    "description": "description",
                    "version": "1.0",
                    "configFormKey": "config form key"
                }
    		}`)
		},
	)

	// -----------------------------------------------------------------------------------
	// fixture to return a private repository and successfully
	// create a commit hook and ssh deploy key and add team
	mux.HandleFunc("/rest/api/1.0/projects/example/repos/team", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"name": "team",
			"slug": "example/team",
			"public": false
		}`)
	})

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/team/settings/hooks/hkey/enabled",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
    			"enabled": false,
	    		"details": {
		    	    "key": "key",
			        "name": "key name",
                    "type": "key type",
                    "description": "description",
                    "version": "1.0",
                    "configFormKey": "config form key"
                }
    		}`)
		},
	)

	mux.HandleFunc(
		"/rest/api/1.0/projects/example/repos/team/settings/hooks/hkey/settings",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
    			"enabled": false,
	    		"details": {
		    	    "key": "key",
			        "name": "key name",
                    "type": "key type",
                    "description": "description",
                    "version": "1.0",
                    "configFormKey": "config form key"
                }
    		}`)
		},
	)

	// Key requests
	mux.HandleFunc("/rest/ssh/1.0/keys", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"values": [{
			"id": 123,
			"test": "key",
			"label": "key label"
		}]}`)
	})
}

func TeardownStashFixtures() {
	dbtest.Teardown()
	stashServer.Close()
}
