package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"testing"

	"github.com/drone/drone/common"
	"github.com/drone/drone/common/ccmenu"
	"github.com/drone/drone/datastore"
	"github.com/drone/drone/datastore/mock"
	"github.com/drone/drone/server/recorder"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

var badgeTests = []struct {
	badge    []byte
	state    string
	activity string
	status   string
	err      error
}{
	{badgeSuccess, common.StateSuccess, "Sleeping", "Success", nil},
	{badgeStarted, common.StateRunning, "Building", "Unknown", nil},
	{badgeError, common.StateError, "Sleeping", "Exception", nil},
	{badgeError, common.StateKilled, "Sleeping", "Exception", nil},
	{badgeFailure, common.StateFailure, "Sleeping", "Failure", nil},
	{badgeNone, "", "", "", datastore.ErrKeyNotFound},
}

func TestBadges(t *testing.T) {
	store := new(mocks.Datastore)
	url_, _ := url.Parse("http://localhost:8080")

	g := Goblin(t)
	g.Describe("Badges", func() {

		g.It("should serve svg badges", func() {
			for _, test := range badgeTests {
				rw := recorder.New()
				ctx := &gin.Context{Engine: gin.Default(), Writer: rw}

				repo := &common.Repo{FullName: "foo/bar"}
				if len(test.state) != 0 {
					repo.Last = &common.Build{State: test.state}
				}

				ctx.Set("datastore", store)
				ctx.Set("repo", repo)

				GetBadge(ctx)
				g.Assert(rw.Code).Equal(200)
				g.Assert(rw.Body.Bytes()).Equal(test.badge)
				g.Assert(rw.HeaderMap.Get("Content-Type")).Equal("image/svg+xml")
			}
		})

		g.It("should serve ccmenu xml", func() {

			for _, test := range badgeTests {
				rw := recorder.New()
				ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
				ctx.Request = &http.Request{URL: url_}

				repo := &common.Repo{FullName: "foo/bar"}
				ctx.Set("datastore", store)
				ctx.Set("repo", repo)

				build := &common.Build{State: test.state}
				store.On("BuildLast", repo.FullName).Return(build, test.err).Once()
				GetCC(ctx)

				// in an error scenario (ie no build exists) we should
				// return a 404 not found error.
				if test.err != nil {
					g.Assert(rw.Status()).Equal(404)
					continue
				}

				// else parse the CCMenu xml output and verify
				// it matches the expected values.
				cc := &ccmenu.CCProjects{}
				xml.Unmarshal(rw.Body.Bytes(), cc)
				g.Assert(cc.Project.Activity).Equal(test.activity)
				g.Assert(cc.Project.LastBuildStatus).Equal(test.status)
				g.Assert(rw.HeaderMap.Get("Content-Type")).Equal("application/xml; charset=utf-8")
			}
		})
	})
}
