package github

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/franela/goblin"
)

func TestHook(t *testing.T) {
	var (
		github Github
		r      *http.Request
		body   *bytes.Buffer
	)

	g := goblin.Goblin(t)

	g.Describe("Hook", func() {
		g.BeforeEach(func() {
			github = Github{}
			body = bytes.NewBuffer([]byte{})
			r, _ = http.NewRequest("POST", "https://drone.com/hook", body)
		})

		g.Describe("For a Pull Request", func() {
			g.BeforeEach(func() {
				r.Header.Set("X-Github-Event", "pull_request")
			})

			g.It("Should set build author to the pull request author", func() {
				hookJson, err := ioutil.ReadFile("fixtures/pull_request.json")
				if err != nil {
					panic(err)
				}
				body.Write(hookJson)

				_, build, err := github.Hook(r)
				g.Assert(err).Equal(nil)
				g.Assert(build.Author).Equal("author")
				g.Assert(build.Avatar).Equal("https://avatars.githubusercontent.com/u/55555?v=3")
			})
		})
	})
}
