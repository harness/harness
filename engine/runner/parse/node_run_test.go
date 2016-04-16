package parse

import (
	"testing"

	"github.com/franela/goblin"
)

func TestRunNode(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("RunNode", func() {
		g.It("should set container name for lookup", func() {
			node0 := NewRunNode()
			node1 := node0.SetName("foo")

			g.Assert(node0.Type().String()).Equal(NodeRun)
			g.Assert(node0.Name).Equal("foo")
			g.Assert(node0).Equal(node1)
		})

		g.It("should fail validation when invalid type", func() {
			node := RunNode{}
			err := node.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Run Node uses an invalid type")
		})

		g.It("should fail validation when invalid name", func() {
			node := NewRunNode()
			err := node.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Run Node has an invalid name")
		})

		g.It("should pass validation", func() {
			node := NewRunNode().SetName("foo")
			g.Assert(node.Validate() == nil).IsTrue()
		})
	})
}
