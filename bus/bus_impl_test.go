package bus

import (
	"sync"
	"testing"

	"github.com/drone/drone/model"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestBus(t *testing.T) {
	g := Goblin(t)
	g.Describe("Event bus", func() {

		g.It("Should unsubscribe", func() {
			c := new(gin.Context)
			b := newEventbus()
			ToContext(c, b)

			c1 := make(chan *Event)
			c2 := make(chan *Event)
			Subscribe(c, c1)
			Subscribe(c, c2)

			g.Assert(len(b.subs)).Equal(2)
		})

		g.It("Should subscribe", func() {
			c := new(gin.Context)
			b := newEventbus()
			ToContext(c, b)

			c1 := make(chan *Event)
			c2 := make(chan *Event)
			Subscribe(c, c1)
			Subscribe(c, c2)

			g.Assert(len(b.subs)).Equal(2)

			Unsubscribe(c, c1)
			Unsubscribe(c, c2)

			g.Assert(len(b.subs)).Equal(0)
		})

		g.It("Should publish", func() {
			c := new(gin.Context)
			b := New()
			ToContext(c, b)

			e1 := NewEvent(Started, &model.Repo{}, &model.Build{}, &model.Job{})
			e2 := NewEvent(Started, &model.Repo{}, &model.Build{}, &model.Job{})
			c1 := make(chan *Event)

			Subscribe(c, c1)

			var wg sync.WaitGroup
			wg.Add(1)

			var r1, r2 *Event
			go func() {
				r1 = <-c1
				r2 = <-c1
				wg.Done()
			}()
			Publish(c, e1)
			Publish(c, e2)
			wg.Wait()
		})
	})

}
