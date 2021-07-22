// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package pubsub

import (
	"context"
	"sync"
	"testing"

	"github.com/drone/drone/core"
)

func TestBus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := newHub()
	events, errc := p.Subscribe(ctx)

	got, err := p.Subscribers()
	if err != nil {
		t.Errorf("Test failed with an error: %s", err.Error())
		return
	}

	if want := 1; got != want {
		t.Errorf("Want %d subscribers, got %d", want, got)
	}

	w := sync.WaitGroup{}
	w.Add(1)
	go func() {
		p.Publish(ctx, new(core.Message))
		p.Publish(ctx, new(core.Message))
		p.Publish(ctx, new(core.Message))
		w.Done()
	}()
	w.Wait()

	w.Add(3)
	go func() {
		for {
			select {
			case <-errc:
				return
			case <-events:
				w.Done()
			}
		}
	}()
	w.Wait()

	cancel()
}
