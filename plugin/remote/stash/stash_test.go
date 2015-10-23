package stash

import (
	"testing"

	"github.com/drone/drone/shared/model"
	"github.com/franela/goblin"
)

func test_Stash(t *testing.T) {
	g := goblin.Goblin(t)
	g.Describe("Stash Plugin", func() {

		g.It("Should return correct kind", func() {
			var s = &Stash{URL: "https://stash.atlassian.com"}
			g.Assert(s.GetKind()).Equal(model.RemoteStash)
		})

		g.It("Should parse hostname", func() {
			var s = &Stash{URL: "https://stash.atlassian.com"}
			g.Assert(s.GetHost()).Equal("stash.atlassian.com")
		})

		g.It("Should authorize user")

		g.It("Should get repos")

		g.It("Should get script/file")

		g.It("Should activate the repo")

		g.It("Should parse commit hook")
	})
}
