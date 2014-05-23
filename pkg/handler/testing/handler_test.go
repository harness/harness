package testing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/drone/drone/pkg/database/testing"
	"github.com/drone/drone/pkg/handler"
	. "github.com/drone/drone/pkg/model"

	"github.com/bmizerany/pat"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRepoHandler(t *testing.T) {
	Setup()
	defer Teardown()

	m := pat.New()

	Convey("Repo Handler", t, func() {
		m.Get("/:host/:owner/:name", handler.RepoHandler(dummyUserRepo))
		Convey("Public repo can be viewed without login", func() {
			req, err := http.NewRequest("GET", "/bitbucket.org/drone/test", nil)
			So(err, ShouldBeNil)
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, req)
			So(rec.Code, ShouldEqual, 200)
		})
		Convey("Public repo can be viewed by another user", func() {
			req, err := http.NewRequest("GET", "/bitbucket.org/drone/test", nil)
			So(err, ShouldBeNil)
			rec := httptest.NewRecorder()
			setUserSession(rec, req, "cavepig@gmail.com")
			m.ServeHTTP(rec, req)
			So(rec.Code, ShouldEqual, 200)
		})

		Convey("Private repo can not be viewed without login", func() {
			req, err := http.NewRequest("GET", "/github.com/drone/drone", nil)
			So(err, ShouldBeNil)
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, req)
			So(rec.Code, ShouldEqual, 303)
		})
		Convey("Private repo can not be viewed by a non team member", func() {
			req, err := http.NewRequest("GET", "/github.com/drone/drone", nil)
			So(err, ShouldBeNil)
			rec := httptest.NewRecorder()
			setUserSession(rec, req, "rick@el.to.ro")
			m.ServeHTTP(rec, req)
			So(rec.Code, ShouldEqual, 404)
		})
	})
}

func dummyUserRepo(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	return handler.RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func setUserSession(w http.ResponseWriter, r *http.Request, username string) {
	handler.SetCookie(w, r, "_sess", username)
	resp := http.Response{Header: w.Header()}
	for _, v := range resp.Cookies() {
		r.AddCookie(v)
	}
}
