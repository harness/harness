package builtin

import (
	"testing"

	"github.com/drone/drone/queue"
	. "github.com/franela/goblin"
)

func TestBuild(t *testing.T) {
	g := Goblin(t)
	g.Describe("Queue", func() {

		g.It("Should publish item", func() {
			w1 := &queue.Work{}
			w2 := &queue.Work{}
			q := New()
			q.Publish(w1)
			q.Publish(w2)
			g.Assert(len(q.items)).Equal(2)
			g.Assert(len(q.itemc)).Equal(2)
		})

		g.It("Should remove item", func() {
			w1 := &queue.Work{}
			w2 := &queue.Work{}
			w3 := &queue.Work{}
			q := New()
			q.Publish(w1)
			q.Publish(w2)
			q.Publish(w3)
			q.Remove(w2)
			g.Assert(len(q.items)).Equal(2)
			g.Assert(len(q.itemc)).Equal(2)
			g.Assert(q.Pull()).Equal(w1)
			g.Assert(q.Pull()).Equal(w3)
		})

		g.It("Should pull item", func() {
			w1 := &queue.Work{}
			w2 := &queue.Work{}
			q := New()
			q.Publish(w1)
			q.Publish(w2)
			g.Assert(q.Pull()).Equal(w1)
			g.Assert(q.Pull()).Equal(w2)
		})

		g.It("Should pull item with ack", func() {
			w := &queue.Work{}
			q := New()
			q.Publish(w)
			g.Assert(q.PullAck()).Equal(w)
			g.Assert(q.acks[w]).Equal(struct{}{})
		})

		g.It("Should ack item", func() {
			w := &queue.Work{}
			q := New()
			q.Publish(w)
			g.Assert(q.PullAck()).Equal(w)
			g.Assert(len(q.acks)).Equal(1)
			g.Assert(q.Ack(w)).Equal(nil)
			g.Assert(len(q.acks)).Equal(0)
		})

		g.It("Should get all items", func() {
			q := New()
			q.Publish(&queue.Work{})
			q.Publish(&queue.Work{})
			q.Publish(&queue.Work{})
			g.Assert(len(q.Items())).Equal(3)
		})
	})
}
