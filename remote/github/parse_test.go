package github

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/github/fixtures"
	"github.com/franela/goblin"
)

func Test_parser(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("GitHub parser", func() {

		g.It("should ignore unsupported hook events", func() {
			buf := bytes.NewBufferString(fixtures.HookPullRequest)
			req, _ := http.NewRequest("POST", "/hook", buf)
			req.Header = http.Header{}
			req.Header.Set(hookEvent, "issues")

			r, b, err := parseHook(req, false)
			g.Assert(r == nil).IsTrue()
			g.Assert(b == nil).IsTrue()
			g.Assert(err == nil).IsTrue()
		})

		g.Describe("given a push hook", func() {
			g.It("should skip when action is deleted", func() {
				raw := []byte(fixtures.HookPushDeleted)
				r, b, err := parsePushHook(raw)
				g.Assert(r == nil).IsTrue()
				g.Assert(b == nil).IsTrue()
				g.Assert(err == nil).IsTrue()
			})
			g.It("should extract repository and build details", func() {
				buf := bytes.NewBufferString(fixtures.HookPush)
				req, _ := http.NewRequest("POST", "/hook", buf)
				req.Header = http.Header{}
				req.Header.Set(hookEvent, hookPush)

				r, b, err := parseHook(req, false)
				g.Assert(err == nil).IsTrue()
				g.Assert(r != nil).IsTrue()
				g.Assert(b != nil).IsTrue()
				g.Assert(b.Event).Equal(model.EventPush)
			})
		})

		g.Describe("given a pull request hook", func() {
			g.It("should skip when action is not open or sync", func() {
				raw := []byte(fixtures.HookPullRequestInvalidAction)
				r, b, err := parsePullHook(raw, false)
				g.Assert(r == nil).IsTrue()
				g.Assert(b == nil).IsTrue()
				g.Assert(err == nil).IsTrue()
			})
			g.It("should skip when state is not open", func() {
				raw := []byte(fixtures.HookPullRequestInvalidState)
				r, b, err := parsePullHook(raw, false)
				g.Assert(r == nil).IsTrue()
				g.Assert(b == nil).IsTrue()
				g.Assert(err == nil).IsTrue()
			})
			g.It("should extract repository and build details", func() {
				buf := bytes.NewBufferString(fixtures.HookPullRequest)
				req, _ := http.NewRequest("POST", "/hook", buf)
				req.Header = http.Header{}
				req.Header.Set(hookEvent, hookPull)

				r, b, err := parseHook(req, false)
				g.Assert(err == nil).IsTrue()
				g.Assert(r != nil).IsTrue()
				g.Assert(b != nil).IsTrue()
				g.Assert(b.Event).Equal(model.EventPull)
			})
		})

		g.Describe("given a deployment hook", func() {
			g.It("should extract repository and build details", func() {
				buf := bytes.NewBufferString(fixtures.HookDeploy)
				req, _ := http.NewRequest("POST", "/hook", buf)
				req.Header = http.Header{}
				req.Header.Set(hookEvent, hookDeploy)

				r, b, err := parseHook(req, false)
				g.Assert(err == nil).IsTrue()
				g.Assert(r != nil).IsTrue()
				g.Assert(b != nil).IsTrue()
				g.Assert(b.Event).Equal(model.EventDeploy)
			})
		})

	})
}
