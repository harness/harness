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

package aigateway

import (
	"context"
	"sync"

	"github.com/harness/gitness/app/store"
)

// counters holds the daily request counters. The planner doesn't know whether they live in
// memory (tests, disabled gateway) or in the database (shared by all replicas).
type counters interface {
	// Increment adds one to the counter unless it already reached limit, atomically.
	Increment(ctx context.Context, day, class string, limit int) (bool, error)
	// Decrement gives one unit back, never going below zero.
	Decrement(ctx context.Context, day, class string) error
	// Raise lifts the counter to at least used.
	Raise(ctx context.Context, day, class string, used int) error
	// Used returns the counter; a missing counter is zero.
	Used(ctx context.Context, day, class string) (int, error)
}

// memCounters keeps the counters in memory.
type memCounters struct {
	mu   sync.Mutex
	used map[[2]string]int
}

func newMemCounters() *memCounters {
	return &memCounters{used: map[[2]string]int{}}
}

func (m *memCounters) Increment(_ context.Context, day, class string, limit int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{day, class}
	if limit <= 0 || m.used[k] >= limit {
		return false, nil
	}
	m.used[k]++
	return true, nil
}

func (m *memCounters) Decrement(_ context.Context, day, class string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{day, class}
	if m.used[k] > 0 {
		m.used[k]--
	}
	return nil
}

func (m *memCounters) Raise(_ context.Context, day, class string, used int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := [2]string{day, class}
	m.used[k] = max(m.used[k], used)
	return nil
}

func (m *memCounters) Used(_ context.Context, day, class string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.used[[2]string{day, class}], nil
}

// storeCounters keeps the counters in the quota table, shared by all replicas.
type storeCounters struct {
	store store.AIQuotaStore
}

func (s storeCounters) Increment(ctx context.Context, day, class string, limit int) (bool, error) {
	return s.store.IncrementQuota(ctx, day, class, limit)
}

func (s storeCounters) Decrement(ctx context.Context, day, class string) error {
	return s.store.DecrementQuota(ctx, day, class)
}

func (s storeCounters) Raise(ctx context.Context, day, class string, used int) error {
	return s.store.RaiseQuota(ctx, day, class, used)
}

func (s storeCounters) Used(ctx context.Context, day, class string) (int, error) {
	q, err := s.store.Find(ctx, day, class)
	if err != nil {
		return 0, err
	}
	return q.Used, nil
}
