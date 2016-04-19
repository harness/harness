package parse

import (
	"testing"

	"github.com/franela/goblin"
)

func TestListNode(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("ListNode", func() {
		g.It("should append nodes", func() {
			node := NewRunNode()

			list0 := NewListNode()
			list1 := list0.Append(node)
			g.Assert(list0.Type().String()).Equal(NodeList)
			g.Assert(list0.Body[0]).Equal(node)
			g.Assert(list0).Equal(list1)
		})

		g.It("should fail validation when invalid type", func() {
			list := ListNode{}
			err := list.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("List Node uses an invalid type")
		})

		g.It("should fail validation when empty body", func() {
			list := NewListNode()
			err := list.Validate()
			g.Assert(err == nil).IsFalse()
			g.Assert(err.Error()).Equal("List Node body is empty")
		})

		g.It("should pass validation", func() {
			node := NewRunNode()
			list := NewListNode()
			list.Append(node)
			g.Assert(list.Validate() == nil).IsTrue()
		})
	})
}
