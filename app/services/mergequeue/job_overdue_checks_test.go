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

package mergequeue

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
)

// errStubFind is returned by stubMergeQueueStore.Find when the test asks it to fail.
var errStubFind = errors.New("find failed")

// stubMergeQueueStore implements store.MergeQueueStore; only Find is wired,
// and it records every requested ID so tests can tell cache hits from misses.
type stubMergeQueueStore struct {
	store.MergeQueueStore
	repoIDs   map[int64]int64 // merge queue ID -> repo ID
	err       error
	findCalls []int64
}

func (s *stubMergeQueueStore) Find(_ context.Context, id int64) (*types.MergeQueue, error) {
	s.findCalls = append(s.findCalls, id)

	if s.err != nil {
		return nil, s.err
	}

	repoID, ok := s.repoIDs[id]
	if !ok {
		return nil, errStubFind
	}

	return &types.MergeQueue{ID: id, RepoID: repoID}, nil
}

func newRepoCacheService(mqStore store.MergeQueueStore) *Service {
	return &Service{mergeQueueStore: mqStore}
}

func TestRepoCacheGet_ZeroValueCacheMiss(t *testing.T) {
	ctx := context.Background()
	mqStore := &stubMergeQueueStore{repoIDs: map[int64]int64{7: 42}}
	svc := newRepoCacheService(mqStore)

	// The zero value must be usable: Get has to allocate the map before writing to it.
	var cache repoCache

	repoID, err := cache.Get(ctx, svc, 7)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if repoID != 42 {
		t.Errorf("repoID = %d, want 42", repoID)
	}

	if want := []int64{7}; !slices.Equal(mqStore.findCalls, want) {
		t.Errorf("findCalls = %v, want %v", mqStore.findCalls, want)
	}

	if got, ok := cache.m[7]; !ok || got != 42 {
		t.Errorf("cache.m[7] = (%d, %t), want (42, true)", got, ok)
	}
}

func TestRepoCacheGet_SecondLookupIsCached(t *testing.T) {
	ctx := context.Background()
	mqStore := &stubMergeQueueStore{repoIDs: map[int64]int64{7: 42}}
	svc := newRepoCacheService(mqStore)

	var cache repoCache

	for i := range 3 {
		repoID, err := cache.Get(ctx, svc, 7)
		if err != nil {
			t.Fatalf("Get() #%d error = %v, want nil", i, err)
		}

		if repoID != 42 {
			t.Errorf("Get() #%d repoID = %d, want 42", i, repoID)
		}
	}

	// Only the first call may reach the store.
	if want := []int64{7}; !slices.Equal(mqStore.findCalls, want) {
		t.Errorf("findCalls = %v, want %v", mqStore.findCalls, want)
	}
}

func TestRepoCacheGet_DistinctQueues(t *testing.T) {
	ctx := context.Background()
	mqStore := &stubMergeQueueStore{repoIDs: map[int64]int64{
		7: 42,
		8: 42, // two queues of the same repo
		9: 43,
	}}
	svc := newRepoCacheService(mqStore)

	var cache repoCache

	// Each ID is requested twice; the second round must be served from the cache.
	// The first lookup of 8 and 9 happens with a non-nil map that lacks the key,
	// which is the branch that falls through to the store.
	for range 2 {
		for _, tt := range []struct {
			qID        int64
			wantRepoID int64
		}{
			{qID: 7, wantRepoID: 42},
			{qID: 8, wantRepoID: 42},
			{qID: 9, wantRepoID: 43},
		} {
			repoID, err := cache.Get(ctx, svc, tt.qID)
			if err != nil {
				t.Fatalf("Get(%d) error = %v, want nil", tt.qID, err)
			}

			if repoID != tt.wantRepoID {
				t.Errorf("Get(%d) repoID = %d, want %d", tt.qID, repoID, tt.wantRepoID)
			}
		}
	}

	if want := []int64{7, 8, 9}; !slices.Equal(mqStore.findCalls, want) {
		t.Errorf("findCalls = %v, want %v", mqStore.findCalls, want)
	}
}

func TestRepoCacheGet_StoreError(t *testing.T) {
	ctx := context.Background()
	mqStore := &stubMergeQueueStore{err: errStubFind}
	svc := newRepoCacheService(mqStore)

	var cache repoCache

	repoID, err := cache.Get(ctx, svc, 7)
	if !errors.Is(err, errStubFind) {
		t.Fatalf("Get() error = %v, want %v", err, errStubFind)
	}

	if repoID != 0 {
		t.Errorf("repoID = %d, want 0", repoID)
	}

	if len(cache.m) != 0 {
		t.Errorf("cache.m = %v, want no entries: a failed lookup must not be cached", cache.m)
	}

	// A failure must not be sticky: once the store recovers, the next call succeeds.
	mqStore.err = nil
	mqStore.repoIDs = map[int64]int64{7: 42}

	repoID, err = cache.Get(ctx, svc, 7)
	if err != nil {
		t.Fatalf("Get() after recovery error = %v, want nil", err)
	}

	if repoID != 42 {
		t.Errorf("repoID after recovery = %d, want 42", repoID)
	}

	if want := []int64{7, 7}; !slices.Equal(mqStore.findCalls, want) {
		t.Errorf("findCalls = %v, want %v", mqStore.findCalls, want)
	}
}
