package queue

import (
	"sync"
	"testing"

	"github.com/drone/drone/model"
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

			w1 := &Work{Job: &model.Job{ID: 0}}
			w2 := &Work{Job: &model.Job{ID: 1}}
			Publish(c, w1)
			Publish(c, w2)
			g.Assert(len(q.items)).Equal(2)
			g.Assert(len(q.itemc)).Equal(2)
		})

		g.It("Should remove item", func() {
			c := new(gin.Context)
			q := newQueue()
			ToContext(c, q)

			w1 := &Work{Job: &model.Job{ID: 0}}
			w2 := &Work{Job: &model.Job{ID: 1}}
			w3 := &Work{Job: &model.Job{ID: 2}}
			Publish(c, w1)
			Publish(c, w2)
			Publish(c, w3)
			Remove(c, w2.Job.ID)
			g.Assert(len(q.items)).Equal(2)
			g.Assert(len(q.itemc)).Equal(2)

			g.Assert(Pull(c)).Equal(w1)
			g.Assert(Pull(c)).Equal(w3)
			g.Assert(Remove(c, w2.Job.ID)).Equal(ErrNotFound)
		})

		g.It("Should pull item", func() {
			c := new(gin.Context)
			q := New()
			ToContext(c, q)

			cn := new(closeNotifier)
			cn.closec = make(chan bool, 1)
			w1 := &Work{Job: &model.Job{ID: 0}}
			w2 := &Work{Job: &model.Job{ID: 1}}

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

		g.It("Should return index", func() {
			c := new(gin.Context)
			q := newQueue()
			ToContext(c, q)

			g.Assert(Length(c)).Equal(0)

			w1 := &Work{Job: &model.Job{ID: 0}}
			w2 := &Work{Job: &model.Job{ID: 1}}

			g.Assert(IndexOf(c, w1.Job.ID)).Equal(-1)
			g.Assert(IndexOf(c, w2.Job.ID)).Equal(-1)

			Publish(c, w1)
			g.Assert(IndexOf(c, w1.Job.ID)).Equal(0)
			g.Assert(IndexOf(c, w2.Job.ID)).Equal(-1)

			Publish(c, w2)
			g.Assert(IndexOf(c, w1.Job.ID)).Equal(0)
			g.Assert(IndexOf(c, w2.Job.ID)).Equal(1)

			Remove(c, w1.Job.ID)
			g.Assert(IndexOf(c, w1.Job.ID)).Equal(-1)
			g.Assert(IndexOf(c, w2.Job.ID)).Equal(0)

			Remove(c, w2.Job.ID)
			g.Assert(IndexOf(c, w1.Job.ID)).Equal(-1)
			g.Assert(IndexOf(c, w2.Job.ID)).Equal(-1)
		})

		g.It("Should return length", func() {
			c := new(gin.Context)
			q := newQueue()
			ToContext(c, q)

			g.Assert(Length(c)).Equal(0)

			w1 := &Work{Job: &model.Job{ID: 0}}

			Publish(c, w1)
			g.Assert(Length(c)).Equal(1)

			w2 := &Work{Job: &model.Job{ID: 1}}
			Publish(c, w2)
			g.Assert(Length(c)).Equal(2)

			Remove(c, w1.Job.ID)
			g.Assert(Length(c)).Equal(1)

			Remove(c, w2.Job.ID)
			g.Assert(Length(c)).Equal(0)
		})
	})
}

type closeNotifier struct {
	closec chan bool
}

func (c *closeNotifier) CloseNotify() <-chan bool {
	return c.closec
}
