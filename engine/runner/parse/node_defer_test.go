package parse

import (
	"testing"

	"github.com/franela/goblin"
)

func TestDeferNode(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("DeferNode", func() {
		g.It("should set body and defer node", func() {
			node0 := NewRunNode()
			node1 := NewRunNode()

			defer0 := NewDeferNode()
			defer1 := defer0.SetBody(node0)
			defer2 := defer0.SetDefer(node1)
			g.Assert(defer0.Type().String()).Equal(NodeDefer)
			g.Assert(defer0.Body).Equal(node0)
			g.Assert(defer0.Defer).Equal(node1)
			g.Assert(defer0).Equal(defer1)
			g.Assert(defer0).Equal(defer2)
		})

		g.It("should fail validation when invalid type", func() {
			defer0 := DeferNode{}
			err := defer0.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Defer Node uses an invalid type")
		})

		g.It("should fail validation when empty body", func() {
			defer0 := NewDeferNode()
			err := defer0.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Defer Node body is empty")
		})

		g.It("should fail validation when empty defer", func() {
			defer0 := NewDeferNode()
			defer0.SetBody(NewRunNode())
			err := defer0.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Defer Node defer is empty")
		})

		g.It("should pass validation", func() {
			defer0 := NewDeferNode()
			defer0.SetBody(NewRunNode())
			defer0.SetDefer(NewRunNode())
			g.Assert(defer0.Validate() == nil).IsTrue()
		})
	})
}
