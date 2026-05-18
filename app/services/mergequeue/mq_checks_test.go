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
	"fmt"
	"testing"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

func pendingEntry(pullReqID int64) *types.MergeQueueEntry {
	return &types.MergeQueueEntry{
		PullReqID:      pullReqID,
		State:          enum.MergeQueueEntryStateChecksPending,
		MergeCommitSHA: sha.Must(fmt.Sprintf("%040d", pullReqID)),
	}
}

func nonPendingEntry(pullReqID int64, state enum.MergeQueueEntryState) *types.MergeQueueEntry {
	return &types.MergeQueueEntry{
		PullReqID:      pullReqID,
		State:          state,
		MergeCommitSHA: sha.Must(fmt.Sprintf("%040d", pullReqID)),
	}
}

func TestUpdateChecks(t *testing.T) {
	const now int64 = 1_000_000

	type wantEntry struct {
		pullReqID         int64
		state             enum.MergeQueueEntryState
		checksCommitSHAOf int64 // pullReqID of the entry whose MergeCommitSHA is the expected ChecksCommitSHA
	}

	tests := []struct {
		name       string
		entries    []*types.MergeQueueEntry
		wantStored []wantEntry // expected entries in toStore, in order
	}{
		{
			name:    "no entries",
			entries: nil,
		},
		{
			name: "no pending entries",
			entries: []*types.MergeQueueEntry{
				nonPendingEntry(1, enum.MergeQueueEntryStateChecksInProgress),
				nonPendingEntry(2, enum.MergeQueueEntryStateMergePending),
			},
		},
		{
			name: "single pending entry",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
			},
			wantStored: []wantEntry{
				{1, enum.MergeQueueEntryStateChecksInProgress, 1},
			},
		},
		{
			name: "two consecutive pending entries",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
			},
			wantStored: []wantEntry{
				{1, enum.MergeQueueEntryStateMergeGroup, 2},
				{2, enum.MergeQueueEntryStateChecksInProgress, 2},
			},
		},
		{
			name: "exactly three pending entries (max chain)",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
				pendingEntry(3),
			},
			wantStored: []wantEntry{
				{1, enum.MergeQueueEntryStateMergeGroup, 3},
				{2, enum.MergeQueueEntryStateMergeGroup, 3},
				{3, enum.MergeQueueEntryStateChecksInProgress, 3},
			},
		},
		{
			name: "four pending entries split into two chains",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
				pendingEntry(3),
				pendingEntry(4),
			},
			wantStored: []wantEntry{
				// chain 1: [1, 2, 3], checksCommitSHA = SHA of 3
				{1, enum.MergeQueueEntryStateMergeGroup, 3},
				{2, enum.MergeQueueEntryStateMergeGroup, 3},
				{3, enum.MergeQueueEntryStateChecksInProgress, 3},
				// chain 2: [4], checksCommitSHA = SHA of 4
				{4, enum.MergeQueueEntryStateChecksInProgress, 4},
			},
		},
		{
			name: "seven pending entries split into three chains",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
				pendingEntry(3),
				pendingEntry(4),
				pendingEntry(5),
				pendingEntry(6),
				pendingEntry(7),
			},
			wantStored: []wantEntry{
				// chain 1: [1, 2, 3]
				{1, enum.MergeQueueEntryStateMergeGroup, 3},
				{2, enum.MergeQueueEntryStateMergeGroup, 3},
				{3, enum.MergeQueueEntryStateChecksInProgress, 3},
				// chain 2: [4, 5, 6]
				{4, enum.MergeQueueEntryStateMergeGroup, 6},
				{5, enum.MergeQueueEntryStateMergeGroup, 6},
				{6, enum.MergeQueueEntryStateChecksInProgress, 6},
				// chain 3: [7]
				{7, enum.MergeQueueEntryStateChecksInProgress, 7},
			},
		},
		{
			name: "six pending entries split at max boundary",
			entries: []*types.MergeQueueEntry{
				pendingEntry(10),
				pendingEntry(20),
				pendingEntry(30),
				pendingEntry(40),
				pendingEntry(50),
				pendingEntry(60),
			},
			wantStored: []wantEntry{
				// chain 1: [10, 20, 30]
				{10, enum.MergeQueueEntryStateMergeGroup, 30},
				{20, enum.MergeQueueEntryStateMergeGroup, 30},
				{30, enum.MergeQueueEntryStateChecksInProgress, 30},
				// chain 2: [40, 50, 60]
				{40, enum.MergeQueueEntryStateMergeGroup, 60},
				{50, enum.MergeQueueEntryStateMergeGroup, 60},
				{60, enum.MergeQueueEntryStateChecksInProgress, 60},
			},
		},
		{
			name: "pending entries separated by non-pending",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
				nonPendingEntry(3, enum.MergeQueueEntryStateChecksInProgress),
				pendingEntry(4),
			},
			wantStored: []wantEntry{
				// chain 1: [1, 2]
				{1, enum.MergeQueueEntryStateMergeGroup, 2},
				{2, enum.MergeQueueEntryStateChecksInProgress, 2},
				// chain 2: [4]
				{4, enum.MergeQueueEntryStateChecksInProgress, 4},
			},
		},
		{
			name: "non-pending at start and end",
			entries: []*types.MergeQueueEntry{
				nonPendingEntry(1, enum.MergeQueueEntryStateMergePending),
				pendingEntry(2),
				pendingEntry(3),
				nonPendingEntry(4, enum.MergeQueueEntryStateMergeGroup),
			},
			wantStored: []wantEntry{
				{2, enum.MergeQueueEntryStateMergeGroup, 3},
				{3, enum.MergeQueueEntryStateChecksInProgress, 3},
			},
		},
	}

	svc := &Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const maximumConsecutiveChecks = 3
			_, toStore := svc.updateChecks(tt.entries, maximumConsecutiveChecks, 0, 0, now)

			if len(toStore) != len(tt.wantStored) {
				t.Fatalf("toStore len = %d, want %d", len(toStore), len(tt.wantStored))
			}

			for i, want := range tt.wantStored {
				got := toStore[i]
				if got.PullReqID != want.pullReqID {
					t.Errorf("toStore[%d]: pullReqID = %d, want %d", i, got.PullReqID, want.pullReqID)
				}
				if got.State != want.state {
					t.Errorf("toStore[%d]: state = %q, want %q", i, got.State, want.state)
				}
				wantChecksCommitSHA := sha.Must(fmt.Sprintf("%040d", want.checksCommitSHAOf))
				if !got.ChecksCommitSHA.Equal(wantChecksCommitSHA) {
					t.Errorf("toStore[%d]: ChecksCommitSHA = %s, want %s",
						i, got.ChecksCommitSHA, wantChecksCommitSHA)
				}
				if got.ChecksStarted == nil {
					t.Errorf("toStore[%d]: ChecksStarted is nil, want %d", i, now)
				} else if *got.ChecksStarted != now {
					t.Errorf("toStore[%d]: ChecksStarted = %d, want %d", i, *got.ChecksStarted, now)
				}
			}
		})
	}
}

func TestUpdateChecks_Concurrency(t *testing.T) {
	const now int64 = 1_000_000

	type wantEntry struct {
		pullReqID         int64
		state             enum.MergeQueueEntryState
		checksCommitSHAOf int64
	}

	tests := []struct {
		name          string
		entries       []*types.MergeQueueEntry
		groupSize     int
		maxInProgress int
		wantStored    []wantEntry
	}{
		{
			name: "concurrency 1: only first chain starts",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
				pendingEntry(3),
				pendingEntry(4),
			},
			groupSize:     3,
			maxInProgress: 1,
			wantStored: []wantEntry{
				{1, enum.MergeQueueEntryStateMergeGroup, 3},
				{2, enum.MergeQueueEntryStateMergeGroup, 3},
				{3, enum.MergeQueueEntryStateChecksInProgress, 3},
			},
		},
		{
			name: "concurrency 2: two chains start",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
				pendingEntry(3),
				pendingEntry(4),
				pendingEntry(5),
				pendingEntry(6),
				pendingEntry(7),
			},
			groupSize:     3,
			maxInProgress: 2,
			wantStored: []wantEntry{
				{1, enum.MergeQueueEntryStateMergeGroup, 3},
				{2, enum.MergeQueueEntryStateMergeGroup, 3},
				{3, enum.MergeQueueEntryStateChecksInProgress, 3},
				{4, enum.MergeQueueEntryStateMergeGroup, 6},
				{5, enum.MergeQueueEntryStateMergeGroup, 6},
				{6, enum.MergeQueueEntryStateChecksInProgress, 6},
			},
		},
		{
			name: "existing in-progress counts toward limit",
			entries: []*types.MergeQueueEntry{
				nonPendingEntry(1, enum.MergeQueueEntryStateChecksInProgress),
				pendingEntry(2),
				pendingEntry(3),
			},
			groupSize:     3,
			maxInProgress: 1,
			wantStored:    nil,
		},
		{
			name: "existing in-progress with room for one more",
			entries: []*types.MergeQueueEntry{
				nonPendingEntry(1, enum.MergeQueueEntryStateChecksInProgress),
				pendingEntry(2),
				pendingEntry(3),
				pendingEntry(4),
				pendingEntry(5),
			},
			groupSize:     3,
			maxInProgress: 2,
			wantStored: []wantEntry{
				{2, enum.MergeQueueEntryStateMergeGroup, 4},
				{3, enum.MergeQueueEntryStateMergeGroup, 4},
				{4, enum.MergeQueueEntryStateChecksInProgress, 4},
			},
		},
		{
			name: "zero maxInProgress means no limit",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
				pendingEntry(3),
				pendingEntry(4),
			},
			groupSize:     3,
			maxInProgress: 0,
			wantStored: []wantEntry{
				{1, enum.MergeQueueEntryStateMergeGroup, 3},
				{2, enum.MergeQueueEntryStateMergeGroup, 3},
				{3, enum.MergeQueueEntryStateChecksInProgress, 3},
				{4, enum.MergeQueueEntryStateChecksInProgress, 4},
			},
		},
		{
			name: "negative maxInProgress means no limit",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				pendingEntry(2),
				pendingEntry(3),
				pendingEntry(4),
			},
			groupSize:     3,
			maxInProgress: -1,
			wantStored: []wantEntry{
				{1, enum.MergeQueueEntryStateMergeGroup, 3},
				{2, enum.MergeQueueEntryStateMergeGroup, 3},
				{3, enum.MergeQueueEntryStateChecksInProgress, 3},
				{4, enum.MergeQueueEntryStateChecksInProgress, 4},
			},
		},
		{
			name: "concurrency allows exactly the existing count",
			entries: []*types.MergeQueueEntry{
				nonPendingEntry(1, enum.MergeQueueEntryStateChecksInProgress),
				nonPendingEntry(2, enum.MergeQueueEntryStateChecksInProgress),
				pendingEntry(3),
			},
			groupSize:     3,
			maxInProgress: 2,
			wantStored:    nil,
		},
		{
			name: "multiple pending groups with gap, concurrency limits second",
			entries: []*types.MergeQueueEntry{
				pendingEntry(1),
				nonPendingEntry(2, enum.MergeQueueEntryStateMergeGroup),
				pendingEntry(3),
			},
			groupSize:     3,
			maxInProgress: 1,
			wantStored: []wantEntry{
				{1, enum.MergeQueueEntryStateChecksInProgress, 1},
			},
		},
	}

	svc := &Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, toStore := svc.updateChecks(tt.entries, tt.groupSize, tt.maxInProgress, 0, now)

			if len(toStore) != len(tt.wantStored) {
				t.Fatalf("toStore len = %d, want %d", len(toStore), len(tt.wantStored))
			}

			for i, want := range tt.wantStored {
				got := toStore[i]
				if got.PullReqID != want.pullReqID {
					t.Errorf("toStore[%d]: pullReqID = %d, want %d", i, got.PullReqID, want.pullReqID)
				}
				if got.State != want.state {
					t.Errorf("toStore[%d]: state = %q, want %q", i, got.State, want.state)
				}
				wantChecksCommitSHA := sha.Must(fmt.Sprintf("%040d", want.checksCommitSHAOf))
				if !got.ChecksCommitSHA.Equal(wantChecksCommitSHA) {
					t.Errorf("toStore[%d]: ChecksCommitSHA = %s, want %s",
						i, got.ChecksCommitSHA, wantChecksCommitSHA)
				}
				if got.ChecksStarted == nil {
					t.Errorf("toStore[%d]: ChecksStarted is nil, want %d", i, now)
				} else if *got.ChecksStarted != now {
					t.Errorf("toStore[%d]: ChecksStarted = %d, want %d", i, *got.ChecksStarted, now)
				}
			}
		})
	}
}

func TestUpdateChecks_Deadline(t *testing.T) {
	const now int64 = 1_000_000

	svc := &Service{}

	tests := []struct {
		name                    string
		maxCheckDurationSeconds int
		wantDeadline            *int64
	}{
		{
			name:                    "zero means no deadline",
			maxCheckDurationSeconds: 0,
			wantDeadline:            nil,
		},
		{
			name:                    "negative means no deadline",
			maxCheckDurationSeconds: -1,
			wantDeadline:            nil,
		},
		{
			name:                    "positive sets deadline",
			maxCheckDurationSeconds: 60,
			wantDeadline:            ptr.Int64(now + 60*1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := []*types.MergeQueueEntry{pendingEntry(1)}
			_, toStore := svc.updateChecks(entries, 3, 0, tt.maxCheckDurationSeconds, now)

			if len(toStore) != 1 {
				t.Fatalf("toStore len = %d, want 1", len(toStore))
			}

			got := toStore[0].ChecksDeadline
			if (tt.wantDeadline == nil) != (got == nil) {
				t.Fatalf("ChecksDeadline nil mismatch: got nil=%v, want nil=%v", got == nil, tt.wantDeadline == nil)
			}
			if tt.wantDeadline != nil && *got != *tt.wantDeadline {
				t.Errorf("ChecksDeadline = %d, want %d", *got, *tt.wantDeadline)
			}
		})
	}
}

func TestUpdateChecks_ConcurrencyLimitHitOnPendingBoundary(t *testing.T) {
	const now int64 = 1_000_000

	svc := &Service{}

	// groupSize=2, maxInProgress=1, entries=[P,P,P,P,P,P]
	// First chain [0,1] is processed, inProgressCount reaches 1.
	// At i=2 (pending), chainStart=2. At i=4 (pending), chainLen=2==groupSize,
	// but the limit is hit and isPending=true, so chainStart is reset to i=4.
	// At i=6 (past end), chainLen=2==groupSize, limit still hit, isPending=false,
	// so chainStart is reset to -1. Only the first chain is stored.
	entries := []*types.MergeQueueEntry{
		pendingEntry(1),
		pendingEntry(2),
		pendingEntry(3),
		pendingEntry(4),
		pendingEntry(5),
		pendingEntry(6),
	}

	_, toStore := svc.updateChecks(entries, 2, 1, 0, now)

	if len(toStore) != 2 {
		t.Fatalf("toStore len = %d, want 2", len(toStore))
	}

	if toStore[0].PullReqID != 1 || toStore[0].State != enum.MergeQueueEntryStateMergeGroup {
		t.Errorf("toStore[0]: got PR %d state %q, want PR 1 MergeGroup",
			toStore[0].PullReqID, toStore[0].State)
	}
	if toStore[1].PullReqID != 2 || toStore[1].State != enum.MergeQueueEntryStateChecksInProgress {
		t.Errorf("toStore[1]: got PR %d state %q, want PR 2 ChecksInProgress",
			toStore[1].PullReqID, toStore[1].State)
	}
}

func TestUpdateChecks_GroupSizeZeroPanics(t *testing.T) {
	svc := &Service{}

	entries := []*types.MergeQueueEntry{pendingEntry(1)}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with groupSize=0, but did not panic")
		}
	}()

	svc.updateChecks(entries, 0, 0, 0, 0)
}

func TestUpdateChecks_UpdatedListReflectsNewStates(t *testing.T) {
	entries := []*types.MergeQueueEntry{
		nonPendingEntry(1, enum.MergeQueueEntryStateMergePending),
		pendingEntry(2),
		pendingEntry(3),
	}

	svc := &Service{}
	updated, _ := svc.updateChecks(entries, 10, 0, 0, 0)

	if len(updated) != len(entries) {
		t.Fatalf("updated len = %d, want %d", len(updated), len(entries))
	}

	want := []enum.MergeQueueEntryState{
		enum.MergeQueueEntryStateMergePending,
		enum.MergeQueueEntryStateMergeGroup,
		enum.MergeQueueEntryStateChecksInProgress,
	}
	for i, e := range updated {
		if e.State != want[i] {
			t.Errorf("updated[%d]: state = %q, want %q", i, e.State, want[i])
		}
	}
}
