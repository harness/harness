package yaml

import (
	"testing"

	"github.com/franela/goblin"
)

func TestLabel(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Label parser", func() {

		g.It("Should parse empty yaml", func() {
			labels := ParseLabelString("")
			g.Assert(len(labels)).Equal(0)
		})

		g.It("Should parse slice", func() {
			labels := ParseLabelString("labels: [foo=bar, baz=boo]")
			g.Assert(len(labels)).Equal(2)
			g.Assert(labels["foo"]).Equal("bar")
			g.Assert(labels["baz"]).Equal("boo")
		})

		g.It("Should parse map", func() {
			labels := ParseLabelString("labels: {foo: bar, baz: boo}")
			g.Assert(labels["foo"]).Equal("bar")
			g.Assert(labels["baz"]).Equal("boo")
		})
	})
}
