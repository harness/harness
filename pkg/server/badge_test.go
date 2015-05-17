package server

import (
	"database/sql"
	"encoding/xml"
	"net/http"
	"net/url"
	"testing"

	"github.com/drone/drone/pkg/ccmenu"
	"github.com/drone/drone/pkg/server/recorder"
	"github.com/drone/drone/pkg/store/mock"
	common "github.com/drone/drone/pkg/types"

	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

var badgeTests = []struct {
	branch   string
	badge    []byte
	state    string
	activity string
	status   string
	err      error
}{
	{"", badgeSuccess, common.StateSuccess, "Sleeping", "Success", nil},
	{"master", badgeSuccess, common.StateSuccess, "Sleeping", "Success", nil},
	{"", badgeStarted, common.StateRunning, "Building", "Unknown", nil},
	{"", badgeError, common.StateError, "Sleeping", "Exception", nil},
	{"", badgeError, common.StateKilled, "Sleeping", "Exception", nil},
	{"", badgeFailure, common.StateFailure, "Sleeping", "Failure", nil},
	{"", badgeNone, "", "", "", sql.ErrNoRows},
}

func TestBadges(t *testing.T) {
	store := new(mocks.Store)
	url_, _ := url.Parse("http://localhost:8080")

	g := Goblin(t)
	g.Describe("Badges", func() {

		g.It("should serve svg badges", func() {
			for _, test := range badgeTests {
				rw := recorder.New()
				ctx := &gin.Context{Engine: gin.Default(), Writer: rw}
				ctx.Request = &http.Request{
					Form: url.Values{},
				}
				if len(test.branch) != 0 {
					ctx.Request.Form.Set("branch", test.branch)
				}

				repo := &common.Repo{FullName: "foo/bar"}
				ctx.Set("datastore", store)
				ctx.Set("repo", repo)

				commit := &common.Commit{State: test.state}
				store.On("CommitLast", repo, test.branch).Return(commit, test.err).Once()
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

				commits := []*common.Commit{
					&common.Commit{State: test.state},
				}
				store.On("CommitList", repo, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(commits, test.err).Once()
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
				g.Assert(rw.Code).Equal(200)
				g.Assert(cc.Project.Activity).Equal(test.activity)
				g.Assert(cc.Project.LastBuildStatus).Equal(test.status)
				g.Assert(rw.HeaderMap.Get("Content-Type")).Equal("application/xml; charset=utf-8")
			}
		})
	})
}
