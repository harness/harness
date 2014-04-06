package event

// oh hey unit tests. or functionality tests, or something.

import (
	"testing"
	"time"
)

func TestAddDelHandler(t *testing.T) {
	r := NewRegistry()

	if len(r.events) != 0 {
		t.Errorf("New registry has non-zero-length event map.")
	}

	h1 := NewHandler(func(ev ...interface{}) {})
	h2 := NewHandler(func(ev ...interface{}) {})

	// Ensure that a handler with no events to handle doesn't get added
	r.AddHandler(h1)
	if len(r.events) != 0 {
		t.Errorf("Adding handler with no events succeded.")
	}

	// Add h1 to a couple of events.
	r.AddHandler(h1, "e1", "E2")
	if len(r.events) != 2 {
		t.Errorf("Adding handler h1 to events failed.")
	}
	if l, ok := r.events["e1"]; !ok || l.Len() != 1 ||
		l.Front().Value != h1 {
		t.Errorf("Handler h1 not added to event e1 correctly.")
	}
	if l, ok := r.events["e2"]; !ok || l.Len() != 1 ||
		l.Front().Value != h1 {
		t.Errorf("Handler h1 not added to event e2 correctly.")
	}

	// Add h2 to a couple of events.
	r.AddHandler(h2, "e2", "E3")
	if len(r.events) != 3 {
		t.Errorf("Adding handler h2 to events failed.")
	}
	if l, ok := r.events["e2"]; !ok || l.Len() != 2 ||
		l.Front().Next().Value.(Handler) != h2 {
		t.Errorf("Handler h2 not added to event e2 correctly.")
	}
	if l, ok := r.events["e3"]; !ok || l.Len() != 1 ||
		l.Front().Value.(Handler) != h2 {
		t.Errorf("Handler h2 not added to event e3 correctly.")
	}

	// Add h1 to some more events, which it may be in already.
	r.AddHandler(h1, "e2", "e3", "e4", "e5")
	if len(r.events) != 5 {
		t.Errorf("Adding handler h1 to more events failed.")
		println(len(r.events))
	}
	if l, ok := r.events["e2"]; !ok || l.Len() != 2 {
		t.Errorf("Handler h1 added twice to event e2.")
	}
	if l, ok := r.events["e3"]; !ok || l.Len() != 2 ||
		l.Front().Next().Value.(Handler) != h1 {
		t.Errorf("Handler h1 not added to event e3 correctly.")
	}

	// Add h2 to a few more events, for testing delete
	r.AddHandler(h2, "e5", "e6")
	if len(r.events) != 6 {
		t.Errorf("Adding handler h2 to more events failed.")
	}

	// Currently, we have the following handlers set up:
	// h1: e1 e2 e3 e4 e5
	// h2:    e2 e3    e5 e6
	// NOTE: for e3, h2 is first and h1 is second in the linked list

	// Delete h1 from a few events. This should remove e4 completely.
	r.DelHandler(h1, "e2", "E3", "e4")
	if _, ok := r.events["e4"]; ok || len(r.events) != 5 {
		t.Errorf("Deleting h1 from some events failed to remove e4.")
	}
	if l, ok := r.events["e2"]; !ok || l.Len() != 1 ||
		l.Front().Value.(Handler) != h2 {
		t.Errorf("Handler h1 not deleted from event e2 correctly.")
	}
	if l, ok := r.events["e3"]; !ok || l.Len() != 1 ||
		l.Front().Value.(Handler) != h2 {
		t.Errorf("Handler h1 not deleted from event e3 correctly.")
	}

	// Now, we have the following handlers set up:
	// h1: e1          e5
	// h2:    e2 e3    e5 e6

	// Delete h2 from a couple of events, removing e2 and e3.
	// Deleting h2 from a handler it is not in should not cause problems.
	r.DelHandler(h2, "e1", "e2", "e3")
	if len(r.events) != 3 {
		t.Errorf("Deleting h2 from some events failed to remove e{2,3}.")
	}
	if l, ok := r.events["e1"]; !ok || l.Len() != 1 ||
		l.Front().Value.(Handler) != h1 {
		t.Errorf("Handler h1 deleted from event e1 incorrectly.")
	}

	// Delete h1 completely.
	r.DelHandler(h1)
	if _, ok := r.events["e1"]; ok || len(r.events) != 2 {
		t.Errorf("Deleting h1 completely failed to remove e1.")
	}
	if l, ok := r.events["e5"]; !ok || l.Len() != 1 ||
		l.Front().Value.(Handler) != h2 {
		t.Errorf("Handler h1 deleted from event e5 incorrectly.")
	}

	// Clear e5 completely
	r.ClearEvents("e5")
	if _, ok := r.events["e5"]; ok || len(r.events) != 1 {
		t.Errorf("Deleting e5 completely failed to remove it.")
	}

	// All that should be left is e6, with h2 as it's only handler.
	if l, ok := r.events["e6"]; !ok || l.Len() != 1 ||
		l.Front().Value.(Handler) != h2 {
		t.Errorf("Remaining event and handler doesn't match expectations.")
	}
}

func TestSimpleDispatch(t *testing.T) {
	r := NewRegistry()
	out := make(chan bool)

	h := NewHandler(func(ev ...interface{}) {
		out <- ev[0].(bool)
	})
	r.AddHandler(h, "send")

	r.Dispatch("send", true)
	if val := <-out; !val {
		t.Fail()
	}

	r.Dispatch("send", false)
	if val := <-out; val {
		t.Fail()
	}
}

func TestParallelDispatch(t *testing.T) {
	r := NewRegistry()
	// ensure we have enough of a buffer that all sends complete
	out := make(chan time.Duration, 5)
	// handler factory :-)
	factory := func(t time.Duration) Handler {
		return NewHandler(func(ev ...interface{}) {
			// t * 10ms sleep
			time.Sleep(10 * t * time.Millisecond)
			out <- t
		})
	}

	// create some handlers and send an event to them
	for _, t := range []time.Duration{5, 11, 2, 15, 8} {
		r.AddHandler(factory(t), "send")
	}
	r.Dispatch("send")

	// If parallel dispatch is working, results from out should be in numerical order
	if val := <-out; val != 2 {
		t.Fail()
	}
	if val := <-out; val != 5 {
		t.Fail()
	}
	if val := <-out; val != 8 {
		t.Fail()
	}
	if val := <-out; val != 11 {
		t.Fail()
	}
	if val := <-out; val != 15 {
		t.Fail()
	}
}

func TestSerialDispatch(t *testing.T) {
	r := NewRegistry()
	r.Serial()
	// ensure we have enough of a buffer that all sends complete
	out := make(chan time.Duration, 5)
	// handler factory :-)
	factory := func(t time.Duration) Handler {
		return NewHandler(func(ev ...interface{}) {
			// t * 10ms sleep
			time.Sleep(10 * t * time.Millisecond)
			out <- t
		})
	}

	// create some handlers and send an event to them
	for _, t := range []time.Duration{5, 11, 2, 15, 8} {
		r.AddHandler(factory(t), "send")
	}
	r.Dispatch("send")

	// If serial dispatch is working, results from out should be in handler order
	if val := <-out; val != 5 {
		t.Fail()
	}
	if val := <-out; val != 11 {
		t.Fail()
	}
	if val := <-out; val != 2 {
		t.Fail()
	}
	if val := <-out; val != 15 {
		t.Fail()
	}
	if val := <-out; val != 8 {
		t.Fail()
	}
}
