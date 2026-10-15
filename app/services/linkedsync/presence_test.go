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

package linkedsync_test

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/linkedsync"
)

type countingPresenceStore struct {
	exists bool
	err    error
	calls  int
}

func (s *countingPresenceStore) ExistsByRootSpace(_ context.Context, _ string) (bool, error) {
	s.calls++
	return s.exists, s.err
}

// The point of the cache is to collapse the firehose down to one query per
// account, so a repeated answer must not re-query.
func TestPresenceCache_CachesPositiveAnswer(t *testing.T) {
	store := &countingPresenceStore{exists: true}
	p := linkedsync.NewPresenceCache(store)
	ctx := context.Background()

	for i := range 3 {
		got, err := p.HasAny(ctx, "acct-1")
		if err != nil {
			t.Fatalf("call %d: HasAny() error = %v", i, err)
		}
		if !got {
			t.Fatalf("call %d: HasAny() = false, want true", i)
		}
	}

	if store.calls != 1 {
		t.Errorf("store queried %d times, want 1", store.calls)
	}
}

// "No linked repos" is the answer for almost every account on the stream, so it
// is the one that most needs caching — an uncached false would leave the query
// running at full stream rate.
func TestPresenceCache_CachesNegativeAnswer(t *testing.T) {
	store := &countingPresenceStore{exists: false}
	p := linkedsync.NewPresenceCache(store)
	ctx := context.Background()

	for i := range 3 {
		got, err := p.HasAny(ctx, "acct-1")
		if err != nil {
			t.Fatalf("call %d: HasAny() error = %v", i, err)
		}
		if got {
			t.Fatalf("call %d: HasAny() = true, want false", i)
		}
	}

	if store.calls != 1 {
		t.Errorf("store queried %d times, want 1", store.calls)
	}
}

func TestPresenceCache_KeysByAccount(t *testing.T) {
	store := &countingPresenceStore{exists: true}
	p := linkedsync.NewPresenceCache(store)
	ctx := context.Background()

	if _, err := p.HasAny(ctx, "acct-1"); err != nil {
		t.Fatalf("HasAny(acct-1): %v", err)
	}
	if _, err := p.HasAny(ctx, "acct-2"); err != nil {
		t.Fatalf("HasAny(acct-2): %v", err)
	}

	if store.calls != 2 {
		t.Errorf("store queried %d times, want 2 (one per account)", store.calls)
	}
}

// A transient database failure must not be remembered as an answer, or one
// blip would blind the consumer to an account for the whole TTL.
func TestPresenceCache_DoesNotCacheErrors(t *testing.T) {
	store := &countingPresenceStore{err: errors.New("db down")}
	p := linkedsync.NewPresenceCache(store)
	ctx := context.Background()

	if _, err := p.HasAny(ctx, "acct-1"); err == nil {
		t.Fatal("HasAny() error = nil, want the store error to propagate")
	}

	store.err = nil
	store.exists = true

	got, err := p.HasAny(ctx, "acct-1")
	if err != nil {
		t.Fatalf("HasAny() after recovery: %v", err)
	}
	if !got {
		t.Error("HasAny() = false after the store recovered, want true")
	}
	if store.calls != 2 {
		t.Errorf("store queried %d times, want 2 (the error must be retried)", store.calls)
	}
}
