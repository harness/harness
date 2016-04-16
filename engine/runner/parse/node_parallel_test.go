package parse

import (
	"testing"

	"github.com/franela/goblin"
)

func TestParallelNode(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("ParallelNode", func() {
		g.It("should append nodes", func() {
			node := NewRunNode()

			parallel0 := NewParallelNode()
			parallel1 := parallel0.Append(node)
			g.Assert(parallel0.Type().String()).Equal(NodeParallel)
			g.Assert(parallel0.Body[0]).Equal(node)
			g.Assert(parallel0).Equal(parallel1)
		})

		g.It("should fail validation when invalid type", func() {
			node := ParallelNode{}
			err := node.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Parallel Node uses an invalid type")
		})

		g.It("should fail validation when empty body", func() {
			node := NewParallelNode()
			err := node.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("Parallel Node body is empty")
		})

		g.It("should pass validation", func() {
			node := NewParallelNode().Append(NewRunNode())
			g.Assert(node.Validate() == nil).IsTrue()
		})
	})
}
