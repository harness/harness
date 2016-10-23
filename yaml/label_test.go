package yaml

import (
	"testing"

	"github.com/franela/goblin"
)

func TestLabel(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Parser", func() {
		g.Describe("Given a yaml file", func() {

			g.It("Should parse a label", func() {
				out := ParseLabelString("label: test")
				g.Assert(out).Equal("test")
			})
		})
	})
}
