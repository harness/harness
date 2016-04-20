package expander

import (
	"testing"

	"github.com/franela/goblin"
)

func TestExpand(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Expand params", func() {

		g.It("Should replace vars with ${key}", func() {
			s := "echo ${FOO} $BAR"
			m := map[string]string{}
			m["FOO"] = "BAZ"
			g.Assert("echo BAZ $BAR").Equal(ExpandString(s, m))
		})

		g.It("Should not replace vars in nil map", func() {
			s := "echo ${FOO} $BAR"
			g.Assert(s).Equal(ExpandString(s, nil))
		})

		g.It("Should escape quoted variables", func() {
			s := `echo "${FOO}"`
			m := map[string]string{}
			m["FOO"] = "hello\nworld"
			g.Assert(`echo "hello\nworld"`).Equal(ExpandString(s, m))
		})

		g.It("Should replace variable prefix", func() {
			s := `tag: ${TAG=${SHA:8}}`
			m := map[string]string{}
			m["TAG"] = ""
			m["SHA"] = "f36cbf54ee1a1eeab264c8e388f386218ab1701b"
			g.Assert("tag: f36cbf54").Equal(ExpandString(s, m))
		})

		g.It("Should handle nested substitution operations", func() {
			s := `echo "${TAG##v}"`
			m := map[string]string{}
			m["TAG"] = "v1.0.0"
			g.Assert(`echo "1.0.0"`).Equal(ExpandString(s, m))
		})
	})
}
