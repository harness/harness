package builtin

import (
	"testing"

	"github.com/drone/drone/eventbus"
	. "github.com/franela/goblin"
)

func TestBuild(t *testing.T) {
	g := Goblin(t)
	g.Describe("Bus", func() {

		g.It("Should unsubscribe", func() {
			c1 := make(chan *eventbus.Event)
			c2 := make(chan *eventbus.Event)
			b := New()
			b.Subscribe(c1)
			b.Subscribe(c2)
			g.Assert(len(b.subs)).Equal(2)
		})

		g.It("Should subscribe", func() {
			c1 := make(chan *eventbus.Event)
			c2 := make(chan *eventbus.Event)
			b := New()
			b.Subscribe(c1)
			b.Subscribe(c2)
			g.Assert(len(b.subs)).Equal(2)
			b.Unsubscribe(c1)
			b.Unsubscribe(c2)
			g.Assert(len(b.subs)).Equal(0)
		})

		g.It("Should send", func() {
			e1 := &eventbus.Event{Name: "foo"}
			e2 := &eventbus.Event{Name: "bar"}
			c := make(chan *eventbus.Event)
			b := New()
			b.Subscribe(c)
			b.Send(e1)
			b.Send(e2)
			r1 := <-c
			r2 := <-c
			g.Assert(e1).Equal(r1)
			g.Assert(e2).Equal(r2)
		})
	})

}
