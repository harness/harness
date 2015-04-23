package server

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/common"
	"github.com/drone/drone/mocks"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

type RecorderImpl struct {
	recorder *httptest.ResponseRecorder
}

func (ri RecorderImpl) Header() http.Header {
	return ri.recorder.Header()
}

func (ri RecorderImpl) Write(buf []byte) (int, error) {
	return ri.recorder.Write(buf)
}

func (ri RecorderImpl) WriteHeader(code int) {
	ri.recorder.WriteHeader(code)
}

func (ri RecorderImpl) CloseNotify() <-chan bool {
	return http.ResponseWriter(ri.recorder).(http.CloseNotifier).CloseNotify()
}

func (ri RecorderImpl) Flush() {
	ri.recorder.Flush()
}

func (ri RecorderImpl) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.ResponseWriter(ri.recorder).(http.Hijacker).Hijack()
}

func (ri RecorderImpl) Size() int {
	return ri.recorder.Body.Len()
}

func (ri RecorderImpl) Status() int {
	return ri.recorder.Code
}

func (ri RecorderImpl) WriteHeaderNow() {
	// no-op?
}

func (ri RecorderImpl) Written() bool {
	return false
}

func TestBadage(t *testing.T) {
	g := Goblin(t)
	g.Describe("Badge", func() {
		// token := common.Token{
		// 	Kind:   "github",
		// 	Login:  "Freya",
		// 	Label:  "github",
		// 	Repos:  []string{"Freya/Hello-World"},
		// 	Scopes: []string{},
		// 	Expiry: 0,
		// 	Issued: 0,
		// }

		var ds *mocks.Datastore
		var ctx gin.Context
		g.BeforeEach(func() {
			router := gin.New()
			ds = new(mocks.Datastore)
			ctx = gin.Context{Engine: router}
			ctx.Set("datastore", ds)
		})

		g.AfterEach(func() {
		})

		g.It("should provide SVG response", func() {
			w := new(RecorderImpl)
			w.recorder = httptest.NewRecorder()
			ctx.Writer = w

			repo := &common.Repo{Owner: "Freya", Name: "Hello-World"}
			ctx.Set("repo", repo)

			// TODO(benschumacher) expand this a lot.
			GetBadge(&ctx)
			g.Assert(w.Status()).Equal(200)

			// Check the variety of states
			for _, state := range []string{common.StateError, common.StateFailure, common.StateKilled, common.StatePending, common.StateRunning, common.StateSuccess} {
				repo.Last = &common.Build{State: state}
				ctx.Set("repo", repo)

				GetBadge(&ctx)
				g.Assert(w.Status()).Equal(200)
			}
		})
		g.It("should provide CCTray response") /*, func() {
			w := httptest.NewRecorder()
			ctx.Writer = w

			repo := &common.Repo{Owner: "Freya", Name: "Hello-World"}
			ctx.Set("repo", repo)

		}*/
	})
}
