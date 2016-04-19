package expander

import (
	"testing"

	"github.com/franela/goblin"
)

func TestSubstitution(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Parameter Substitution", func() {

		g.It("Should substitute simple parameters", func() {
			before := "echo ${GREETING} WORLD"
			after := "echo HELLO WORLD"
			g.Assert(substitute(before, "GREETING", "HELLO")).Equal(after)
		})

		g.It("Should substitute quoted parameters", func() {
			before := "echo \"${GREETING}\" WORLD"
			after := "echo \"HELLO\" WORLD"
			g.Assert(substituteQ(before, "GREETING", "HELLO")).Equal(after)
		})

		g.It("Should substitute parameters and trim prefix", func() {
			before := "echo ${GREETING##asdf} WORLD"
			after := "echo HELLO WORLD"
			g.Assert(substitutePrefix(before, "GREETING", "asdfHELLO")).Equal(after)
		})

		g.It("Should substitute parameters and trim suffix", func() {
			before := "echo ${GREETING%%asdf} WORLD"
			after := "echo HELLO WORLD"
			g.Assert(substituteSuffix(before, "GREETING", "HELLOasdf")).Equal(after)
		})

		g.It("Should substitute parameters without using the default", func() {
			before := "echo ${GREETING=HOLA} WORLD"
			after := "echo HELLO WORLD"
			g.Assert(substituteDefault(before, "GREETING", "HELLO")).Equal(after)
		})

		g.It("Should substitute parameters using the a default", func() {
			before := "echo ${GREETING=HOLA} WORLD"
			after := "echo HOLA WORLD"
			g.Assert(substituteDefault(before, "GREETING", "")).Equal(after)
		})

		g.It("Should substitute parameters with replacement", func() {
			before := "echo ${GREETING/HE/A} MONDE"
			after := "echo ALLO MONDE"
			g.Assert(substituteReplace(before, "GREETING", "HELLO")).Equal(after)
		})

		g.It("Should substitute parameters with left substr", func() {
			before := "echo ${FOO:4} IS COOL"
			after := "echo THIS IS COOL"
			g.Assert(substituteLeft(before, "FOO", "THIS IS A REALLY LONG STRING")).Equal(after)
		})

		g.It("Should substitute parameters with substr", func() {
			before := "echo ${FOO:8:5} IS COOL"
			after := "echo DRONE IS COOL"
			g.Assert(substituteSubstr(before, "FOO", "THIS IS DRONE CI")).Equal(after)
		})
	})
}
