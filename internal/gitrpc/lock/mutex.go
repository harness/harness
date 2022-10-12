// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import (
	"context"
	"errors"
	"sync"
)

// Mutex is basic locker implementation
// do not use it in prod and distributed environment.
type Mutex struct {
	mux   sync.RWMutex
	locks map[string]*Lock
}

func (c *Mutex) AcquireLock(ctx context.Context, key string) (*Lock, error) {
	if c == nil {
		return nil, errors.New("mutex not initialized")
	}

	if c.locks == nil {
		c.locks = make(map[string]*Lock)
	}

	c.mux.RLock()
	defer c.mux.RUnlock()

	lock, ok := c.locks[key]
	if !ok {
		lock = &Lock{
			state:    false,
			key:      key,
			lockChan: make(chan struct{}, 1),
		}
	}
	// TODO: One acquire having to wait causes all to wait?
	select {
	case lock.lockChan <- struct{}{}:
		lock.state = true
		c.locks[key] = lock
		return lock, nil
	case <-ctx.Done():
		return nil, errors.New("deadline exceeded, lock not created")
	}
}

// Lock represents an obtained, app wide lock.
type Lock struct {
	state    bool
	key      string
	lockChan chan struct{}
}

// Key returns the redis key used by the lock.
func (l *Lock) Key() string {
	if l == nil {
		return ""
	}
	return l.key
}

// Locked returns if this key is locked.
func (l *Lock) Locked() bool {
	if l == nil {
		return false
	}
	return l.state
}

// Release manually releases the lock.
func (l *Lock) Release() {
	if l == nil {
		return
	}
	<-l.lockChan
}
