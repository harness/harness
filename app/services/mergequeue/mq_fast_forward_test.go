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

	gitnesserrors "github.com/harness/gitness/errors"
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

func TestIsFastForwardError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil-error",
			err:  nil,
			want: false,
		},
		{
			name: "plain-error",
			err:  errors.New("some other failure"),
			want: false,
		},
		{
			name: "bare-fast-forward-error",
			err:  fastForwardError{},
			want: true,
		},
		{
			name: "fast-forward-error-with-internal",
			err:  fastForwardError{internal: errors.New("non-fast-forward")},
			want: true,
		},
		{
			// The self-heal branch in handlerCheckFinished only fires if detection
			// survives fmt.Errorf("%w") wrapping, which is how fastForward returns it.
			name: "wrapped-fast-forward-error",
			err:  fmt.Errorf("failed to fast-forward: %w", fastForwardError{internal: errors.New("boom")}),
			want: true,
		},
		{
			name: "double-wrapped-fast-forward-error",
			err: fmt.Errorf("outer: %w",
				fmt.Errorf("inner: %w", fastForwardError{})),
			want: true,
		},
		{
			// A precondition-failed status error on its own must NOT be treated as a
			// fast-forward error - only the typed wrapper produced by fastForward is.
			name: "precondition-failed-not-wrapped",
			err:  gitnesserrors.PreconditionFailed("non fast forward"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFastForwardError(tt.err); got != tt.want {
				t.Errorf("isFastForwardError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestFastForwardErrorMessage(t *testing.T) {
	const base = "failed to fast-forward merge queue branch to merge commit"

	t.Run("without-internal", func(t *testing.T) {
		got := fastForwardError{}.Error()
		if got != base {
			t.Errorf("want %q, got %q", base, got)
		}
	})

	t.Run("with-internal", func(t *testing.T) {
		got := fastForwardError{internal: errors.New("non-fast-forward")}.Error()
		want := base + ": non-fast-forward"
		if got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	})
}
