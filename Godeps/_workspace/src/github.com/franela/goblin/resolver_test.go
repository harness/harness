package goblin

import (
	"testing"
)

func TestResolver(t *testing.T) {
	g := Goblin(t)

	g.Describe("Resolver", func() {
		g.It("Should resolve the stack until the test", func() {
			dummyFunc(g)
		})
	})
}

func dummyFunc(g *G) {
	stack := ResolveStack(1)
	g.Assert(len(stack)).Equal(3)
}
