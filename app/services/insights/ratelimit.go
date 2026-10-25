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

package insights

import (
	"math"
	"sync"
	"time"
)

// bucket is a token bucket refilled continuously.
type bucket struct {
	tokens float64
	last   time.Time
}

// rateLimiter is an in-memory token bucket per project. The capacity equals
// the per-minute limit, so a project may burst a full minute of events.
// State is per process; with several replicas the effective limit is multiplied.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[int64]*bucket
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{buckets: map[int64]*bucket{}}
}

// allow tries to take n tokens from the bucket of the project. limitPerMinute
// <= 0 disables limiting. On denial it returns how long to wait until n tokens
// are available (at least one second).
func (l *rateLimiter) allow(projectID int64, limitPerMinute, n int, now time.Time) (bool, time.Duration) {
	if limitPerMinute <= 0 || n <= 0 {
		return true, 0
	}

	capacity := float64(limitPerMinute)
	perSecond := capacity / 60

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[projectID]
	if !ok {
		b = &bucket{tokens: capacity, last: now}
		l.buckets[projectID] = b
	}

	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		b.tokens = math.Min(capacity, b.tokens+elapsed*perSecond)
	}
	b.last = now
	b.tokens = math.Min(b.tokens, capacity) // the limit might have been lowered

	need := float64(n)
	if need > capacity {
		need = capacity // an envelope bigger than the limit can still pass once the bucket is full
	}
	if b.tokens >= need {
		b.tokens -= need
		return true, 0
	}

	wait := time.Duration((need - b.tokens) / perSecond * float64(time.Second))
	if wait < time.Second {
		wait = time.Second
	}
	return false, wait
}

// forget drops the state of a project.
func (l *rateLimiter) forget(projectID int64) {
	l.mu.Lock()
	delete(l.buckets, projectID)
	l.mu.Unlock()
}
