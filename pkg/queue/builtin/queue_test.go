package builtin

import (
	"sync"
	"testing"

	"github.com/drone/drone/pkg/queue"
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
			g.Assert(q.Remove(w2)).Equal(ErrNotFound)
		})

		g.It("Should pull item", func() {
			w1 := &queue.Work{}
			w2 := &queue.Work{}
			q := New()
			c := new(closeNotifier)
			q.Publish(w1)
			q.Publish(w2)
			g.Assert(q.Pull()).Equal(w1)
			g.Assert(q.PullClose(c)).Equal(w2)
			g.Assert(q.acks[w1]).Equal(struct{}{})
			g.Assert(q.acks[w2]).Equal(struct{}{})
			g.Assert(len(q.acks)).Equal(2)
		})

		g.It("Should cancel pulling item", func() {
			q := New()
			c := new(closeNotifier)
			c.closec = make(chan bool, 1)
			var wg sync.WaitGroup
			go func() {
				wg.Add(1)
				g.Assert(q.PullClose(c) == nil).IsTrue()
				wg.Done()
			}()
			go func() {
				c.closec <- true
			}()
			wg.Wait()
		})

		g.It("Should ack item", func() {
			w := &queue.Work{}
			c := new(closeNotifier)
			q := New()
			q.Publish(w)
			g.Assert(q.PullClose(c)).Equal(w)
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

type closeNotifier struct {
	closec chan bool
}

func (c *closeNotifier) CloseNotify() <-chan bool {
	return c.closec
}
