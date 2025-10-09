// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	if config.DelayFunc == nil {
		config.DelayFunc = func(_ int) time.Duration {
			return config.RetryDelay
		}
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
		return nil, NewError(ErrorKindGenerateTokenFailed, key, nil)
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
func (m *inMemMutex) Key() string {
	return m.key
}

// Lock acquires the lock. It fails with error if the lock is already held.
func (m *inMemMutex) Lock(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isHeld {
		return NewError(ErrorKindLockHeld, m.key, nil)
	}

	if m.provider.acquire(m.key, m.token, m.expiry) {
		m.isHeld = true
		return nil
	}

	timeout := time.NewTimer(m.waitTime)
	defer timeout.Stop()

	for i := 1; !m.isHeld && i <= m.tries; i++ {
		if err := m.retry(ctx, i, timeout); err != nil {
			return err
		}
	}
	return nil
}

func (m *inMemMutex) retry(ctx context.Context, attempt int, timeout *time.Timer) error {
	if m.isHeld {
		return nil
	}
	if attempt == m.tries {
		return NewError(ErrorKindMaxRetriesExceeded, m.key, nil)
	}

	delay := time.NewTimer(m.delayFunc(attempt))
	defer delay.Stop()

	select {
	case <-ctx.Done():
		return NewError(ErrorKindContext, m.key, ctx.Err())
	case <-timeout.C:
		return NewError(ErrorKindCannotLock, m.key, nil)
	case <-delay.C: // just wait
	}

	if m.provider.acquire(m.key, m.token, m.expiry) {
		m.isHeld = true
	}
	return nil
}

// Unlock releases the lock. It fails with error if the lock is not currently held.
func (m *inMemMutex) Unlock(_ context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isHeld || !m.provider.release(m.key, m.token) {
		return NewError(ErrorKindLockNotHeld, m.key, nil)
	}

	m.isHeld = false
	return nil
}

func randstr(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buffer), nil
}
