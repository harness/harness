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
	"errors"
	"time"
)

// ErrInvalidTTL is returned for a non-positive ttl, which redis treats as "never expires".
var ErrInvalidTTL = errors.New("idempotency: ttl must be positive")

// State is the outcome of a Claim.
type State int

const (
	// StateClaimed means the caller now owns the key and has to do the operation.
	StateClaimed State = iota
	// StateInFlight means somebody else claimed the key and hasn't committed yet.
	StateInFlight
	// StateDone means the operation was already completed, so the caller can skip it.
	StateDone
)

func (s State) String() string {
	switch s {
	case StateClaimed:
		return "claimed"
	case StateInFlight:
		return "in-flight"
	case StateDone:
		return "done"
	default:
		return "unknown"
	}
}

// Store remembers which operations have already been done.
// A claim is short lived while the operation runs and is turned into a long lived record by Commit.
type Store interface {
	// Claim atomically takes the key for the given ttl. It fails with ErrInvalidTTL if ttl is not positive.
	Claim(ctx context.Context, key string, ttl time.Duration) (State, error)
	// Commit records the key as done for the given ttl, even if the claim already expired.
	Commit(ctx context.Context, key string, ttl time.Duration) error
	// Release removes a claimed key, so the operation can be retried immediately.
	Release(ctx context.Context, key string) error
}
