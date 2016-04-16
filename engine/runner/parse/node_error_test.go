package parse

import (
	"testing"

	"github.com/franela/goblin"
)

func TestErrorNode(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("ErrorNode", func() {
		g.It("should set body and error node", func() {
			node0 := NewRunNode()
			node1 := NewRunNode()

			error0 := NewErrorNode()
			error1 := error0.SetBody(node0)
			error2 := error0.SetDefer(node1)
			g.Assert(error0.Type().String()).Equal(NodeError)
			g.Assert(error0.Body).Equal(node0)
			g.Assert(error0.Defer).Equal(node1)
			g.Assert(error0).Equal(error1)
			g.Assert(error0).Equal(error2)
		})

		g.It("should fail validation when invalid type", func() {
			error0 := ErrorNode{}
			err := error0.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Error Node uses an invalid type")
		})

		g.It("should fail validation when empty body", func() {
			error0 := NewErrorNode()
			err := error0.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Error Node body is empty")
		})

		g.It("should fail validation when empty error", func() {
			error0 := NewErrorNode()
			error0.SetBody(NewRunNode())
			err := error0.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Error Node defer is empty")
		})

		g.It("should pass validation", func() {
			error0 := NewErrorNode()
			error0.SetBody(NewRunNode())
			error0.SetDefer(NewRunNode())
			g.Assert(error0.Validate() == nil).IsTrue()
		})
	})
}
