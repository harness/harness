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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
			return
		}
		err = mx.Unlock(context.Background())
		require.NoError(t, err)
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
	err = mx.Unlock(context.Background())
	require.NoError(t, err)
	wg.Wait()
}

func Test_inMemMutex_MaxTries(t *testing.T) {
	manager := NewInMemory(Config{
		App:        "gitness",
		Namespace:  "pullreq",
		Expiry:     1 * time.Second,
		Tries:      2,
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

		err = mx.Lock(context.Background())
		if err == nil {
			t.Errorf("error should be returned while locking %s instead of nil", mx.Key())
			return
		}
		var errLock *Error
		if !errors.As(err, &errLock) {
			t.Errorf("expected error lock.Error, got: %v", err)
			return
		}
		if errLock.Kind != ErrorKindMaxRetriesExceeded {
			t.Errorf("expected lock.MaxRetriesExceeded, got: %v", err)
			return
		}
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
	err = mx.Unlock(context.Background())
	require.NoError(t, err)
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
