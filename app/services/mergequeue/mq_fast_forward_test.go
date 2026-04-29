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
	"errors"
	"fmt"
	"testing"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
)

func makeSHA(id int64) sha.SHA {
	return sha.Must(fmt.Sprintf("%040d", id))
}

func completeEntry(pullReqID int64, orderIndex int64) *types.MergeQueueEntry {
	return &types.MergeQueueEntry{
		PullReqID:      pullReqID,
		OrderIndex:     orderIndex,
		BaseCommitSHA:  makeSHA(pullReqID * 100),
		MergeCommitSHA: makeSHA(pullReqID*100 + 1),
	}
}

func TestFindEntriesToMerge(t *testing.T) {
	s := &Service{}

	tests := []struct {
		name      string
		entry     *types.MergeQueueEntry
		entries   []*types.MergeQueueEntry
		wantCount int
		wantErr   bool
		wantErrIs error
	}{
		{
			name:  "single-entry-match",
			entry: completeEntry(1, 10),
			entries: []*types.MergeQueueEntry{
				completeEntry(1, 10),
			},
			wantCount: 1,
		},
		{
			name:  "match-at-end-returns-all",
			entry: completeEntry(3, 30),
			entries: []*types.MergeQueueEntry{
				completeEntry(1, 10),
				completeEntry(2, 20),
				completeEntry(3, 30),
			},
			wantCount: 3,
		},
		{
			name:  "match-in-middle-returns-prefix",
			entry: completeEntry(2, 20),
			entries: []*types.MergeQueueEntry{
				completeEntry(1, 10),
				completeEntry(2, 20),
				completeEntry(3, 30),
			},
			wantCount: 2,
		},
		{
			name:  "match-at-start-returns-one",
			entry: completeEntry(1, 10),
			entries: []*types.MergeQueueEntry{
				completeEntry(1, 10),
				completeEntry(2, 20),
				completeEntry(3, 30),
			},
			wantCount: 1,
		},
		{
			name:  "no-match-returns-error",
			entry: completeEntry(99, 99),
			entries: []*types.MergeQueueEntry{
				completeEntry(1, 10),
				completeEntry(2, 20),
			},
			wantErr:   true,
			wantErrIs: errIncompleteMergeQueue,
		},
		{
			name:      "empty-entries-returns-error",
			entry:     completeEntry(1, 10),
			entries:   []*types.MergeQueueEntry{},
			wantErr:   true,
			wantErrIs: errIncompleteMergeQueue,
		},
		{
			name:  "empty-base-sha-before-match-returns-error",
			entry: completeEntry(2, 20),
			entries: []*types.MergeQueueEntry{
				{
					PullReqID:      1,
					OrderIndex:     10,
					BaseCommitSHA:  sha.SHA{}, // empty
					MergeCommitSHA: makeSHA(101),
				},
				completeEntry(2, 20),
			},
			wantErr:   true,
			wantErrIs: errIncompleteMergeQueue,
		},
		{
			name:  "empty-merge-sha-before-match-returns-error",
			entry: completeEntry(2, 20),
			entries: []*types.MergeQueueEntry{
				{
					PullReqID:      1,
					OrderIndex:     10,
					BaseCommitSHA:  makeSHA(100),
					MergeCommitSHA: sha.SHA{}, // empty
				},
				completeEntry(2, 20),
			},
			wantErr:   true,
			wantErrIs: errIncompleteMergeQueue,
		},
		{
			name:  "empty-sha-after-match-not-reached",
			entry: completeEntry(1, 10),
			entries: []*types.MergeQueueEntry{
				completeEntry(1, 10),
				{
					PullReqID:      2,
					OrderIndex:     20,
					BaseCommitSHA:  sha.SHA{}, // empty - but not reached
					MergeCommitSHA: sha.SHA{},
				},
			},
			wantCount: 1,
		},
		{
			name:  "duplicate-order-index-matches-first",
			entry: completeEntry(2, 10),
			entries: []*types.MergeQueueEntry{
				completeEntry(1, 10),
				completeEntry(2, 10),
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.findEntriesToMerge(tt.entry, tt.entries)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("want error %v, got %v", tt.wantErrIs, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.wantCount {
				t.Errorf("want %d entries, got %d", tt.wantCount, len(result))
			}
		})
	}
}
