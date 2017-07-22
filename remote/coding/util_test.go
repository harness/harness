package coding

import (
	"testing"

	"github.com/franela/goblin"
)

func Test_util(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Coding util", func() {

		g.It("Should form project full name", func() {
			g.Assert(projectFullName("gk", "prj")).Equal("gk/prj")
		})
	})
}
