// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package pubsub

import (
	"testing"

	"github.com/drone/drone/core"
)

func nop(*core.Message) {}

func TestSubscription_publish(t *testing.T) {
	s := &subscriber{
		handler: make(chan *core.Message, 5),
		quit:    make(chan struct{}),
	}

	e := new(core.Message)
	s.publish(e)

	if got, want := len(s.handler), 1; got != want {
		t.Errorf("Want buffered channel size %d, got %d", want, got)
	}
	if got, want := <-s.handler, e; got != want {
		t.Errorf("Want event received from channel")
	}
	if got, want := len(s.handler), 0; got != want {
		t.Errorf("Want buffered channel size %d, got %d", want, got)
	}
}

func TestSubscription_buffer(t *testing.T) {
	s := &subscriber{
		handler: make(chan *core.Message, 1),
		quit:    make(chan struct{}),
	}

	// the buffer size is 1 to simulate what happens
	// if the subscriber cannot keep up with processing
	// and the buffer fills up. In this case, events
	// should be ignored until pending events are
	// processed.

	e := new(core.Message)
	s.publish(e)
	s.publish(e)
	s.publish(e)
	s.publish(e)
	s.publish(e)

	if got, want := len(s.handler), 1; got != want {
		t.Errorf("Want buffered channel size %d, got %d", want, got)
	}
}

func TestSubscription_stop(t *testing.T) {
	s := &subscriber{
		handler: make(chan *core.Message, 1),
		quit:    make(chan struct{}),
	}

	if got, want := s.done, false; got != want {
		t.Errorf("Want subscription open")
	}

	s.close()
	if got, want := s.done, true; got != want {
		t.Errorf("Want subscription closed")
	}

	// if the subscription is closed we should
	// ignore any new events being published.

	e := new(core.Message)
	s.publish(e)
	s.publish(e)
	s.publish(e)
	s.publish(e)
	s.publish(e)
}
