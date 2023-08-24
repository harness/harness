// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import (
	"context"
	"sync"
	"testing"
	"time"
)

func Test_inMemMutex_Lock(t *testing.T) {
	manager := NewInMemory(Config{
		App:        "gitness",
		Namespace:  "pullreq",
		Expiry:     3 * time.Second,
		Tries:      10,
		RetryDelay: 300 * time.Millisecond,
	})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		mx, err := manager.NewMutex("key1")
		if err != nil {
			t.Errorf("mutex not created, err: %v", err)
			return
		}
		if err := mx.Lock(context.Background()); err != nil {
			t.Errorf("error from go routine while locking %s, err: %v", mx.Key(), err)
		}
		mx.Unlock(context.Background())
	}()

	mx, err := manager.NewMutex("key1")
	if err != nil {
		t.Errorf("mutex not created, err: %v", err)
		return
	}
	if err := mx.Lock(context.Background()); err != nil {
		t.Errorf("error while locking %v, err: %v", mx.Key(), err)
	}
	time.Sleep(1 * time.Second)
	mx.Unlock(context.Background())
	wg.Wait()
}

func Test_inMemMutex_LockAndWait(t *testing.T) {
	wg := &sync.WaitGroup{}
	manager := NewInMemory(Config{
		App:        "gitness",
		Namespace:  "pullreq",
		Expiry:     3 * time.Second,
		Tries:      10,
		RetryDelay: 300 * time.Millisecond,
	})
	fn := func(n int) {
		mx, err := manager.NewMutex("Key1")
		if err != nil {
			t.Errorf("mutex not created routine %d, err: %v", n, err)
			return
		}
		defer func() {
			if err := mx.Unlock(context.Background()); err != nil {
				t.Errorf("failed to unlock %d", n)
			}
			wg.Done()
		}()
		if err := mx.Lock(context.Background()); err != nil {
			t.Errorf("failed to lock %d", n)
		}
		time.Sleep(50 * time.Millisecond)
	}

	wg.Add(3)
	go fn(1)
	go fn(2)
	go fn(3)
	wg.Wait()
}
