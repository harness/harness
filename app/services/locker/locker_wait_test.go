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

package locker

import (
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/lock"
)

func newTestLocker() *Locker {
	return NewLocker(lock.NewInMemory(lock.Config{
		App:           "test",
		Expiry:        10 * time.Second,
		Tries:         8,
		RetryDelay:    250 * time.Millisecond,
		DriftFactor:   0.01,
		TimeoutFactor: 0.25,
	}))
}

func TestLockBranch_DefaultFailsFast(t *testing.T) {
	l := newTestLocker()
	unlock, err := l.LockBranch(context.Background(), 1, "main", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	start := time.Now()
	if _, err = l.LockBranch(context.Background(), 1, "main", 10*time.Second); err == nil {
		t.Fatal("expected the second lock to fail")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("default wait took too long: %s", time.Since(start))
	}
}

func TestLockBranch_WithWaitQueues(t *testing.T) {
	l := newTestLocker()
	unlock, err := l.LockBranch(context.Background(), 1, "main", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// hold longer than the default budget (about 2s) but release well within the requested wait.
	go func() {
		time.Sleep(3 * time.Second)
		unlock()
	}()

	unlock2, err := l.LockBranch(context.Background(), 1, "main", 10*time.Second, WithWait(10*time.Second))
	if err != nil {
		t.Fatalf("expected to queue for the lock: %v", err)
	}
	unlock2()
}
