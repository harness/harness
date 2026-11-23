// Copyright 2026 Harness, Inc.
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

// Package idempotency provides stores that remember which operations were already performed.
package idempotency

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryStore is a Store for single instance deployments.
type InMemoryStore struct {
	mx        sync.Mutex
	entries   map[string]inMemEntry
	nextSweep time.Time
}

type inMemEntry struct {
	expires time.Time
	done    bool
}

// inMemorySweepInterval is how often expired entries are dropped.
const inMemorySweepInterval = time.Minute

func NewInMemory() *InMemoryStore {
	return &InMemoryStore{entries: make(map[string]inMemEntry)}
}

func (s *InMemoryStore) Claim(_ context.Context, key string, ttl time.Duration) (State, error) {
	if ttl <= 0 {
		return StateClaimed, fmt.Errorf("failed to claim key %q: %w", key, ErrInvalidTTL)
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	now := time.Now()
	if now.After(s.nextSweep) {
		for k, entry := range s.entries {
			if now.After(entry.expires) {
				delete(s.entries, k)
			}
		}
		s.nextSweep = now.Add(inMemorySweepInterval)
	}

	if entry, ok := s.entries[key]; ok && !now.After(entry.expires) {
		if entry.done {
			return StateDone, nil
		}
		return StateInFlight, nil
	}
	s.entries[key] = inMemEntry{expires: now.Add(ttl)}

	return StateClaimed, nil
}

func (s *InMemoryStore) Commit(_ context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("failed to commit key %q: %w", key, ErrInvalidTTL)
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	s.entries[key] = inMemEntry{expires: time.Now().Add(ttl), done: true}

	return nil
}

func (s *InMemoryStore) Release(_ context.Context, key string) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	delete(s.entries, key)

	return nil
}
