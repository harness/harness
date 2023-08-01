// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// InMemory is a local implementation of a MutexManager that it's intended to be used during development.
type InMemory struct {
	config Config // force value copy
	mutex  sync.Mutex
	keys   map[string]inMemEntry
}

// NewInMemory creates a new InMemory instance only used for development.
func NewInMemory(config Config) *InMemory {
	keys := make(map[string]inMemEntry)

	return &InMemory{
		config: config,
		keys:   keys,
	}
}

// NewMutex creates a mutex for the given key. The returned mutex is not held
// and must be acquired with a call to .Lock.
func (m *InMemory) NewMutex(key string, options ...Option) (Mutex, error) {
	var (
		token string
		err   error
	)

	// copy default values
	config := m.config

	// set default delayFunc
	config.DelayFunc = func(i int) time.Duration {
		return config.RetryDelay
	}

	// override config with custom options
	for _, opt := range options {
		opt.Apply(&config)
	}

	// format key
	key = formatKey(config.App, config.Namespace, key)

	switch {
	case config.Value != "":
		token = config.Value
	case config.GenValueFunc != nil:
		token, err = config.GenValueFunc()
	default:
		token, err = randstr(32)
	}
	if err != nil {
		return nil, NewError(GenerateTokenFailed, key, nil)
	}

	// waitTime logic is similar to redis implementation:
	// https://github.com/go-redsync/redsync/blob/e1e5da6654c81a2069d6a360f1a31c21f05cd22d/mutex.go#LL81C4-L81C100
	waitTime := config.Expiry
	if config.TimeoutFactor > 0 {
		waitTime = time.Duration(int64(float64(config.Expiry) * config.TimeoutFactor))
	}

	lock := inMemMutex{
		expiry:    config.Expiry,
		waitTime:  waitTime,
		tries:     config.Tries,
		delayFunc: config.DelayFunc,
		provider:  m,
		key:       key,
		token:     token,
	}

	return &lock, nil
}

func (m *InMemory) acquire(key, token string, ttl time.Duration) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()

	entry, ok := m.keys[key]
	if ok && entry.validUntil.After(now) {
		return false
	}

	m.keys[key] = inMemEntry{token, now.Add(ttl)}

	return true
}

func (m *InMemory) release(key, token string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	entry, ok := m.keys[key]
	if !ok || entry.token != token {
		return false
	}

	delete(m.keys, key)

	return true
}

type inMemEntry struct {
	token      string
	validUntil time.Time
}

type inMemMutex struct {
	mutex sync.Mutex // Used while manipulating the internal state of the lock itself

	provider *InMemory

	expiry   time.Duration
	waitTime time.Duration

	tries     int
	delayFunc DelayFunc

	key    string
	token  string // A random string used to safely release the lock
	isHeld bool
}

// Key returns the key to be locked.
func (l *inMemMutex) Key() string {
	return l.key
}

// Lock acquires the lock. It fails with error if the lock is already held.
func (l *inMemMutex) Lock(ctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.isHeld {
		return NewError(LockHeld, l.key, nil)
	}

	if l.provider.acquire(l.key, l.token, l.expiry) {
		l.isHeld = true
		return nil
	}

	timeout := time.NewTimer(l.waitTime)
	defer timeout.Stop()

	for i := 1; i <= l.tries; i++ {
		select {
		case <-ctx.Done():
			return NewError(Context, l.key, ctx.Err())
		case <-timeout.C:
			return NewError(CannotLock, l.key, nil)
		case <-time.After(l.delayFunc(i)):
			if l.provider.acquire(l.key, l.token, l.expiry) {
				l.isHeld = true
				return nil
			}
		}
	}
	return NewError(MaxRetriesExceeded, l.key, nil)
}

// Unlock releases the lock. It fails with error if the lock is not currently held.
func (l *inMemMutex) Unlock(_ context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.isHeld || !l.provider.release(l.key, l.token) {
		return NewError(LockNotHeld, l.key, nil)
	}

	l.isHeld = false
	return nil
}

func randstr(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buffer), nil
}
