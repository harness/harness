package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/drone/drone/common"
	"github.com/drone/drone/common/ccmenu"
	"github.com/drone/drone/datastore"
	"github.com/drone/drone/mocks"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestBadge(t *testing.T) {
	g := Goblin(t)
	g.Describe("Badge", func() {
		var ctx gin.Context
		owner := "Freya"
		name := "Hello-World"
		fullName := owner + "/" + name
		repo := &common.Repo{Owner: owner, Name: name, FullName: fullName}
		g.BeforeEach(func() {
			ctx = gin.Context{Engine: gin.Default()}
			url, _ := url.Parse("http://drone.local/badges/" + fullName)
			ctx.Request = &http.Request{URL: url}
			ctx.Set("repo", repo)
		})

		g.AfterEach(func() {
		})

		cycleStateTester := func(expector gin.HandlerFunc, handle gin.HandlerFunc, validator func(state string, w *ResponseRecorder)) {
			for idx, state := range []string{"", common.StateError, common.StateFailure, common.StateKilled, common.StatePending, common.StateRunning, common.StateSuccess} {
				w := NewResponseRecorder()
				ctx.Writer = w

				repo.Last = &common.Build{
					Started:  time.Now().UTC().Unix(),
					Finished: time.Now().UTC().Unix(),
					Number:   idx,
					State:    state,
				}
				ctx.Set("repo", repo)

				if expector != nil {
					expector(&ctx)
				}

				handle(&ctx)

				validator(state, w)
			}
		}

		g.It("should provide SVG response", func() {
			{
				// 1. verify no "last" build
				w := NewResponseRecorder()
				ctx.Writer = w
				ctx.Request.URL.Path += "/status.svg"

				GetBadge(&ctx)

				g.Assert(w.Status()).Equal(200)
				g.Assert(w.HeaderMap.Get("content-type")).Equal("image/svg+xml")
				g.Assert(strings.Contains(w.Body.String(), ">none")).IsTrue()
			}

			// 2. verify a variety of "last" build states
			cycleStateTester(nil, GetBadge, func(state string, w *ResponseRecorder) {
				g.Assert(w.Status()).Equal(200)
				g.Assert(w.HeaderMap.Get("content-type")).Equal("image/svg+xml")

				// this may be excessive, but does effectively verify behavior
				switch state {
				case common.StateSuccess:
					g.Assert(strings.Contains(w.Body.String(), ">success")).IsTrue()
				case common.StatePending, common.StateRunning:
					g.Assert(strings.Contains(w.Body.String(), ">started")).IsTrue()
				case common.StateError, common.StateKilled:
					g.Assert(strings.Contains(w.Body.String(), ">error")).IsTrue()
				case common.StateFailure:
					g.Assert(strings.Contains(w.Body.String(), ">failure")).IsTrue()
				default:
					g.Assert(strings.Contains(w.Body.String(), ">none")).IsTrue()
				}
			})
		})

		g.It("should provide CCTray response", func() {
			{
				// 1. verify no "last" build
				w := NewResponseRecorder()
				ctx.Writer = w
				ctx.Request.URL.Path += "/cc.xml"

				ds := new(mocks.Datastore)
				ctx.Set("datastore", ds)

				ds.On("BuildLast", fullName).Return(nil, datastore.ErrKeyNotFound).Once()
				GetCC(&ctx)

				g.Assert(w.Status()).Equal(404)
			}

			// 2. verify a variety of "last" build states
			cycleStateTester(func(c *gin.Context) {
				repo := ToRepo(c)
				ds := new(mocks.Datastore)
				ctx.Set("datastore", ds)
				ds.On("BuildLast", fullName).Return(repo.Last, nil).Once()
			},
				GetCC,
				func(state string, w *ResponseRecorder) {
					g.Assert(w.Status()).Equal(200)

					v := ccmenu.CCProjects{}
					xml.Unmarshal(w.Body.Bytes(), &v)
					switch state {
					case common.StateSuccess:
						g.Assert(v.Project.Activity).Equal("Sleeping")
						g.Assert(v.Project.LastBuildStatus).Equal("Success")
					case common.StatePending, common.StateRunning:
						g.Assert(v.Project.Activity).Equal("Building")
						g.Assert(v.Project.LastBuildStatus).Equal("Unknown")
					case common.StateError, common.StateKilled:
						g.Assert(v.Project.Activity).Equal("Sleeping")
						g.Assert(v.Project.LastBuildStatus).Equal("Exception")
					case common.StateFailure:
						g.Assert(v.Project.Activity).Equal("Sleeping")
						g.Assert(v.Project.LastBuildStatus).Equal("Failure")
					default:
						g.Assert(v.Project.Activity).Equal("Sleeping")
						g.Assert(v.Project.LastBuildStatus).Equal("Unknown")
					}
				})
		})
	})
}
