package engine

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestBus(t *testing.T) {
	g := Goblin(t)
	g.Describe("Event bus", func() {

		g.It("Should unsubscribe", func() {
			c1 := make(chan *Event)
			c2 := make(chan *Event)
			b := newEventbus()
			b.subscribe(c1)
			b.subscribe(c2)
			g.Assert(len(b.subs)).Equal(2)
		})

		g.It("Should subscribe", func() {
			c1 := make(chan *Event)
			c2 := make(chan *Event)
			b := newEventbus()
			b.subscribe(c1)
			b.subscribe(c2)
			g.Assert(len(b.subs)).Equal(2)
			b.unsubscribe(c1)
			b.unsubscribe(c2)
			g.Assert(len(b.subs)).Equal(0)
		})

		g.It("Should send", func() {
			em := map[string]bool{"foo": true, "bar": true}
			e1 := &Event{Name: "foo"}
			e2 := &Event{Name: "bar"}
			c := make(chan *Event)
			b := newEventbus()
			b.subscribe(c)
			b.send(e1)
			b.send(e2)
			r1 := <-c
			r2 := <-c
			g.Assert(em[r1.Name]).Equal(true)
			g.Assert(em[r2.Name]).Equal(true)
		})
	})

}
