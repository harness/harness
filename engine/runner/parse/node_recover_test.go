package parse

import (
	"testing"

	"github.com/franela/goblin"
)

func TestRecoverNode(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("RecoverNode", func() {
		g.It("should set body", func() {
			node0 := NewRunNode()

			recover0 := NewRecoverNode()
			recover1 := recover0.SetBody(node0)
			g.Assert(recover0.Type().String()).Equal(NodeRecover)
			g.Assert(recover0.Body).Equal(node0)
			g.Assert(recover0).Equal(recover1)
		})

		g.It("should fail validation when invalid type", func() {
			recover0 := RecoverNode{}
			err := recover0.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Recover Node uses an invalid type")
		})

		g.It("should fail validation when empty body", func() {
			recover0 := NewRecoverNode()
			err := recover0.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Recover Node body is empty")
		})

		g.It("should pass validation", func() {
			recover0 := NewRecoverNode()
			recover0.SetBody(NewRunNode())
			g.Assert(recover0.Validate() == nil).IsTrue()
		})
	})
}
