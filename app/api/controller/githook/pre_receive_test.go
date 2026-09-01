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

package githook

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRepoID int64 = 42
const testSpaceID int64 = 7

// rootSpaceStorageCall records the arguments of a single RootSpaceStorage call.
type rootSpaceStorageCall struct {
	spaceID       int64
	additionalKiB int64
}

// limiterMock is a limiter.ResourceLimiter that returns preconfigured errors and
// records the calls it received.
type limiterMock struct {
	repoSizeErr         error
	rootSpaceStorageErr error

	repoSizeCalls         []int64
	rootSpaceStorageCalls []rootSpaceStorageCall
}

func (l *limiterMock) RepoCount(context.Context, int64, int) error {
	return nil
}

func (l *limiterMock) RepoSize(_ context.Context, repoID int64) error {
	l.repoSizeCalls = append(l.repoSizeCalls, repoID)
	return l.repoSizeErr
}

func (l *limiterMock) RootSpaceStorage(_ context.Context, spaceID int64, additionalKiB int64) error {
	l.rootSpaceStorageCalls = append(
		l.rootSpaceStorageCalls,
		rootSpaceStorageCall{spaceID: spaceID, additionalKiB: additionalKiB},
	)
	return l.rootSpaceStorageErr
}

var _ limiter.ResourceLimiter = (*limiterMock)(nil)

// repoIDCacheMock serves a single repository by its ID.
type repoIDCacheMock struct {
	repo *types.RepositoryCore
}

func (repoIDCacheMock) Stats() (int64, int64) { return 0, 0 }

func (repoIDCacheMock) Evict(context.Context, int64) {}

func (c repoIDCacheMock) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if c.repo == nil || c.repo.ID != id {
		return nil, gitness_store.ErrResourceNotFound
	}
	return c.repo, nil
}

func newTestController(repo *types.RepositoryCore, l limiter.ResourceLimiter) *Controller {
	return &Controller{
		repoFinder: refcache.NewRepoFinder(
			nil,
			nil,
			repoIDCacheMock{repo: repo},
			nil,
			storecache.Evictor[*types.RepositoryCore]{},
		),
		limiter: l,
	}
}

func newTestRepo() *types.RepositoryCore {
	return &types.RepositoryCore{
		ID:            testRepoID,
		ParentID:      testSpaceID,
		Identifier:    "repo",
		Path:          "space/repo",
		GitUID:        "git-uid",
		DefaultBranch: "main",
		State:         enum.RepoStateActive,
		Type:          enum.RepoTypeNormal,
	}
}

// TestPreReceive_StorageLimits covers how PreReceive translates the limiter
// results into hook output: hard limits block the push with a user facing
// message, soft limits only warn, and unexpected failures bubble up as errors.
func TestPreReceive_StorageLimits(t *testing.T) {
	unexpectedErr := errors.New("boom")

	tests := []struct {
		name                string
		repoSizeErr         error
		rootSpaceStorageErr error
		// repoState is used to force an early exit that preserves output.Messages.
		repoState        enum.RepoState
		opType           enum.GitOpType
		expectedErr      string
		expectedOutErr   *string
		expectedMessages []string
		// expectRootStorageCall is false when the repo size check short circuits.
		expectRootStorageCall bool
	}{
		{
			name:                  "within limits",
			repoState:             enum.RepoStateActive,
			opType:                enum.GitOpTypeAPIRefsOnly,
			expectRootStorageCall: true,
		},
		{
			name:                  "repo size hard limit blocks push",
			repoSizeErr:           limiter.ErrMaxRepoSizeReached,
			repoState:             enum.RepoStateActive,
			opType:                enum.GitOpTypeAPIRefsOnly,
			expectedOutErr:        strPtr(limiter.ErrMaxRepoSizeReached.Error()),
			expectRootStorageCall: false,
		},
		{
			name: "repo size hard limit blocks push when wrapped",
			repoSizeErr: fmt.Errorf(
				"repo 42: %w", limiter.ErrMaxRepoSizeReached,
			),
			repoState:             enum.RepoStateActive,
			opType:                enum.GitOpTypeAPIRefsOnly,
			expectedOutErr:        strPtr("repo 42: " + limiter.ErrMaxRepoSizeReached.Error()),
			expectRootStorageCall: false,
		},
		{
			name:        "repo size soft limit only warns",
			repoSizeErr: limiter.ErrRepoSizeSoftLimitReached,
			// Archived repos are rejected after the limiter checks, which keeps the
			// accumulated warning in the returned output.
			repoState: enum.RepoStateArchived,
			opType:    enum.GitOpTypeGitPush,
			expectedOutErr: strPtr(
				fmt.Sprintf("Push not allowed when repository is in '%s' state", enum.RepoStateArchived),
			),
			expectedMessages:      []string{limiter.ErrRepoSizeSoftLimitReached.Error(), ""},
			expectRootStorageCall: true,
		},
		{
			name:                  "unexpected repo size error is returned",
			repoSizeErr:           unexpectedErr,
			repoState:             enum.RepoStateActive,
			opType:                enum.GitOpTypeAPIRefsOnly,
			expectedErr:           "failed to check repository size limit",
			expectRootStorageCall: false,
		},
		{
			name:                  "total storage limit blocks push",
			rootSpaceStorageErr:   limiter.ErrMaxTotalStorageReached,
			repoState:             enum.RepoStateActive,
			opType:                enum.GitOpTypeAPIRefsOnly,
			expectedOutErr:        strPtr(limiter.ErrMaxTotalStorageReached.Error()),
			expectRootStorageCall: true,
		},
		{
			name:                  "unexpected total storage error is returned",
			rootSpaceStorageErr:   unexpectedErr,
			repoState:             enum.RepoStateActive,
			opType:                enum.GitOpTypeAPIRefsOnly,
			expectedErr:           "failed to check total storage limit",
			expectRootStorageCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &limiterMock{
				repoSizeErr:         tt.repoSizeErr,
				rootSpaceStorageErr: tt.rootSpaceStorageErr,
			}

			repo := newTestRepo()
			repo.State = tt.repoState

			out, err := newTestController(repo, l).PreReceive(
				context.Background(),
				nil,
				nil,
				types.GithookPreReceiveInput{
					GithookInputBase: types.GithookInputBase{
						RepoID:        testRepoID,
						OperationType: tt.opType,
					},
				},
			)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				// the original error must stay in the chain for logging/debugging
				assert.ErrorIs(t, err, unexpectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedOutErr, out.Error)
			assert.Equal(t, tt.expectedMessages, out.Messages)

			assert.Equal(t, []int64{testRepoID}, l.repoSizeCalls)
			if tt.expectRootStorageCall {
				// the total storage quota is enforced against the repo's parent space
				assert.Equal(t,
					[]rootSpaceStorageCall{{spaceID: testSpaceID, additionalKiB: 0}},
					l.rootSpaceStorageCalls,
				)
			} else {
				assert.Empty(t, l.rootSpaceStorageCalls)
			}
		})
	}
}

// TestPreReceive_StorageLimitsSkipped documents the operations that must not be
// subject to storage quotas because they never add content.
func TestPreReceive_StorageLimitsSkipped(t *testing.T) {
	tests := []struct {
		name     string
		repoType enum.RepoType
		opType   enum.GitOpType
	}{
		{
			name:     "merge queue fast forward",
			repoType: enum.RepoTypeNormal,
			opType:   enum.GitOpTypeMergeQueue,
		},
		{
			name:     "linked repository sync",
			repoType: enum.RepoTypeLinked,
			opType:   enum.GitOpTypeAPILinkedSync,
		},
		{
			name:     "repository management",
			repoType: enum.RepoTypeNormal,
			opType:   enum.GitOpTypeManageRepo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// any limiter call would return a blocking error, so an empty output
			// proves the limiter wasn't consulted
			l := &limiterMock{
				repoSizeErr:         limiter.ErrMaxRepoSizeReached,
				rootSpaceStorageErr: limiter.ErrMaxTotalStorageReached,
			}

			repo := newTestRepo()
			repo.Type = tt.repoType

			out, err := newTestController(repo, l).PreReceive(
				context.Background(),
				nil,
				nil,
				types.GithookPreReceiveInput{
					GithookInputBase: types.GithookInputBase{
						RepoID:        testRepoID,
						OperationType: tt.opType,
					},
				},
			)
			require.NoError(t, err)

			assert.Empty(t, l.repoSizeCalls)
			assert.Empty(t, l.rootSpaceStorageCalls)

			if tt.opType == enum.GitOpTypeManageRepo {
				// management operations are rejected before the repo is even loaded
				require.NotNil(t, out.Error)
				return
			}
			assert.Nil(t, out.Error)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
