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

package pullreq

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/git/sha"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	mocksse "github.com/harness/gitness/mocks/sse"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/mock"
)

// TestCloseBecauseNonUniqueMergeBase verifies that the PR gets closed at the new source SHA,
// and that the recorded pusher is dropped only when the source SHA actually changes.
func TestCloseBecauseNonUniqueMergeBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		sourceSHA     string
		wantPushedBy  *int64
		activityError error
	}{
		{
			name:         "source SHA changed, pusher is cleared",
			sourceSHA:    branchUpdateNewSHA,
			wantPushedBy: nil,
		},
		{
			name:         "source SHA unchanged, pusher is kept",
			sourceSHA:    branchUpdateOldSHA,
			wantPushedBy: ptr.Int64(7),
		},
		{
			name:          "best-effort failures are ignored",
			sourceSHA:     branchUpdateNewSHA,
			wantPushedBy:  nil,
			activityError: errors.New("db error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pullreqStore := &mockstore.PullReqStore{}
			repoStore := &mockstore.RepoStore{}
			activityStore := &mockstore.PullReqActivityStore{}
			sseStreamer := &mocksse.Streamer{}

			pr := makeBranchUpdatePR(enum.PullReqSubStateNone)
			pr.SourceSHAPushedBy = ptr.Int64(7)

			repo := &types.Repository{ID: 1, NumOpenPulls: 1}
			repoStore.On("Find", int64(1)).Return(repo, nil).Once()
			repoStore.On("UpdateOptLock", repo, mock.Anything).
				Return(repo, test.activityError).Once()

			pullreqStore.On("UpdateOptLock", pr, mock.Anything).
				Run(func(args mock.Arguments) {
					mutateFn, _ := args.Get(1).(func(pr *types.PullReq) error)
					if err := mutateFn(pr); err != nil {
						t.Errorf("unexpected mutate error: %v", err)
					}
				}).Return(pr, nil).Once()

			activityStore.On("CreateWithPayload", pr, mock.Anything, mock.Anything, mock.Anything).
				Return((*types.PullReqActivity)(nil), test.activityError).Twice()

			sseStreamer.On("Publish", int64(1), enum.SSETypePullReqUpdated, pr).Once()

			svc := &Service{
				repoStore:         repoStore,
				pullreqStore:      pullreqStore,
				activityStore:     activityStore,
				sseStreamer:       sseStreamer,
				pullreqEvReporter: mockpullreq.NewStubReporter(t),
			}

			sourceSHA := sha.Must(test.sourceSHA)
			if err := svc.CloseBecauseNonUniqueMergeBase(context.Background(), defaultSHA, sourceSHA, pr); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if pr.State != enum.PullReqStateClosed {
				t.Errorf("state = %q, want %q", pr.State, enum.PullReqStateClosed)
			}
			if pr.SourceSHA != test.sourceSHA {
				t.Errorf("source SHA = %q, want %q", pr.SourceSHA, test.sourceSHA)
			}
			switch {
			case test.wantPushedBy == nil && pr.SourceSHAPushedBy != nil:
				t.Errorf("source SHA pushed by = %d, want nil", *pr.SourceSHAPushedBy)
			case test.wantPushedBy != nil &&
				(pr.SourceSHAPushedBy == nil || *pr.SourceSHAPushedBy != *test.wantPushedBy):
				t.Errorf("source SHA pushed by = %v, want %d", pr.SourceSHAPushedBy, *test.wantPushedBy)
			}

			pullreqStore.AssertExpectations(t)
			repoStore.AssertExpectations(t)
			activityStore.AssertExpectations(t)
			sseStreamer.AssertExpectations(t)
		})
	}
}

// TestCloseBecauseNonUniqueMergeBase_NotOpen verifies that a PR that isn't open is left alone.
func TestCloseBecauseNonUniqueMergeBase_NotOpen(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}

	pr := makeBranchUpdatePR(enum.PullReqSubStateNone)
	pr.State = enum.PullReqStateMerged

	repoStore.On("Find", int64(1)).Return(&types.Repository{ID: 1}, nil).Once()
	pullreqStore.On("UpdateOptLock", pr, mock.Anything).
		Return((*types.PullReq)(nil), ErrPullReqNotOpen).
		Run(func(args mock.Arguments) {
			mutateFn, _ := args.Get(1).(func(pr *types.PullReq) error)
			if err := mutateFn(pr); !errors.Is(err, ErrPullReqNotOpen) {
				t.Errorf("mutate error = %v, want %v", err, ErrPullReqNotOpen)
			}
		}).Once()

	svc := &Service{repoStore: repoStore, pullreqStore: pullreqStore}

	err := svc.CloseBecauseNonUniqueMergeBase(context.Background(), defaultSHA, sha.Must(branchUpdateNewSHA), pr)
	if !errors.Is(err, ErrPullReqNotOpen) {
		t.Fatalf("error = %v, want %v", err, ErrPullReqNotOpen)
	}

	pullreqStore.AssertExpectations(t)
	repoStore.AssertExpectations(t)
}
