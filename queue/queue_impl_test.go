package queue

import (
	"sync"
	"testing"

	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestBuild(t *testing.T) {
	g := Goblin(t)
	g.Describe("Queue", func() {

		g.It("Should publish item", func() {
			c := new(gin.Context)
			q := newQueue()
			ToContext(c, q)

			w1 := &Work{}
			w2 := &Work{}
			Publish(c, w1)
			Publish(c, w2)
			g.Assert(len(q.items)).Equal(2)
			g.Assert(len(q.itemc[DefaultLabel])).Equal(2)
		})

		g.It("Should remove item", func() {
			c := new(gin.Context)
			q := newQueue()
			ToContext(c, q)

			w1 := &Work{}
			w2 := &Work{}
			w3 := &Work{}
			Publish(c, w1)
			Publish(c, w2)
			Publish(c, w3)
			Remove(c, w2)
			g.Assert(len(q.items)).Equal(2)
			g.Assert(len(q.itemc[DefaultLabel])).Equal(2)

			g.Assert(Pull(c)).Equal(w1)
			g.Assert(Pull(c)).Equal(w3)
			g.Assert(Remove(c, w2)).Equal(ErrNotFound)
		})

		g.It("Should pull item", func() {
			c := new(gin.Context)
			q := New()
			ToContext(c, q)

			cn := new(closeNotifier)
			cn.closec = make(chan bool, 1)
			w1 := &Work{}
			w2 := &Work{}

			Publish(c, w1)
			g.Assert(Pull(c)).Equal(w1)

			Publish(c, w2)
			g.Assert(PullClose(c, cn)).Equal(w2)
		})

		g.It("Should cancel pulling item", func() {
			c := new(gin.Context)
			q := New()
			ToContext(c, q)

			cn := new(closeNotifier)
			cn.closec = make(chan bool, 1)
			var wg sync.WaitGroup
			go func() {
				wg.Add(1)
				g.Assert(PullClose(c, cn) == nil).IsTrue()
				wg.Done()
			}()
			go func() {
				cn.closec <- true
			}()
			wg.Wait()

		})

		g.It("Should cancel pulling item with labels", func() {
			c := new(gin.Context)
			q := New()
			ToContext(c, q)

			w1 := &Work{}
			w1.Label = "1"
			Publish(c, w1)

			cn := new(closeNotifier)
			cn.closec = make(chan bool, 1)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				//g.Assert(PullClose(c, cn) == nil).IsTrue()
				g.Assert(PullCloseWithLabels(c, []string{"2"}, cn) == nil).IsTrue()
				wg.Done()
			}()
			go func() {
				cn.closec <- true
			}()
			wg.Wait()

		})

		g.It("Should pulling with label", func() {
			c := new(gin.Context)
			q := New()
			ToContext(c, q)

			cn := new(closeNotifier)
			cn.closec = make(chan bool, 1)
			w1 := &Work{}
			w1.Label = "1"
			w2 := &Work{}
			w2.Label = "2"
			w3 := &Work{}
			w3.Label = "3"
			w4 := &Work{}
			w4.Label = "2"
			w5 := &Work{}
			w5.Label = "4"

			Publish(c, w1)
			Publish(c, w2)
			Publish(c, w3)
			Publish(c, w4)
			g.Assert(PullWithLabels(c, []string{"2"})).Equal(w2)
			g.Assert(PullWithLabels(c, []string{"2"})).Equal(w4)
			g.Assert(PullWithLabels(c, []string{"2", "3"})).Equal(w3)
			g.Assert(PullWithLabels(c, []string{"2", "1"})).Equal(w1)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				g.Assert(PullWithLabels(c, []string{"2", "4"})).Equal(w5)
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				Publish(c, w5)
				wg.Done()
			}()
			wg.Wait()
		})
	})
}

type closeNotifier struct {
	closec chan bool
}

func (c *closeNotifier) CloseNotify() <-chan bool {
	return c.closec
}
