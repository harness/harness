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
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type infoEntryStore struct {
	store.MergeQueueEntryStore

	entries []*types.MergeQueueEntry
	err     error
}

func (f infoEntryStore) ListForMergeQueue(context.Context, int64) ([]*types.MergeQueueEntry, error) {
	return f.entries, f.err
}

type infoQueueStore struct {
	store.MergeQueueStore

	queue *types.MergeQueue
	err   error
}

func (f infoQueueStore) FindByRepoAndBranch(context.Context, int64, string) (*types.MergeQueue, error) {
	return f.queue, f.err
}

type infoCheckStore struct {
	store.CheckStore

	checks map[string][]types.Check
	err    error
}

func (f infoCheckStore) List(
	_ context.Context, _ int64, commitSHA string, _ types.CheckListOptions,
) ([]types.Check, error) {
	return f.checks[commitSHA], f.err
}

type infoPullReqStore struct {
	store.PullReqStore

	prs map[int64]*types.PullReq
}

func (f infoPullReqStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	pr, ok := f.prs[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return pr, nil
}

const checksSHA = "3333333333333333333333333333333333333333"

func infoEntry(prID int64, checksCommit sha.SHA) *types.MergeQueueEntry {
	return &types.MergeQueueEntry{
		PullReqID:       prID,
		MergeQueueID:    1,
		State:           enum.MergeQueueEntryStateChecksInProgress,
		MergeMethod:     enum.MergeMethodSquash,
		ChecksCommitSHA: checksCommit,
	}
}

func TestBuildMergeQueueInfo(t *testing.T) {
	repo := &types.RepositoryCore{ID: 5}
	setup := protection.MergeQueueSetup{RequiredChecks: []string{"build", "lint"}}
	withChecks := sha.Must(checksSHA)

	first, second := infoEntry(10, sha.None), infoEntry(11, withChecks)

	t.Run("reports the position and merges reported with required checks", func(t *testing.T) {
		svc := &Service{
			mergeQueueEntryStore: infoEntryStore{entries: []*types.MergeQueueEntry{first, second}},
			checkStore: infoCheckStore{checks: map[string][]types.Check{
				checksSHA: {
					{Identifier: "test", Status: enum.CheckStatusSuccess},
					{Identifier: "build", Status: enum.CheckStatusRunning},
				},
			}},
		}

		info, err := svc.BuildMergeQueueInfo(t.Context(), repo, second, setup)
		require.NoError(t, err)
		require.Equal(t, 1, info.PullRequestsAhead)
		require.Equal(t, enum.MergeQueueEntryStateChecksInProgress, info.State)
		require.Equal(t, enum.MergeMethodSquash, info.MergeMethod)
		require.Equal(t, withChecks, info.ChecksCommitSHA)

		// sorted by identifier; "lint" is required but not reported yet, "test" is reported but not required
		require.Len(t, info.Checks, 3)
		require.Equal(t, "build", info.Checks[0].Check.Identifier)
		require.True(t, info.Checks[0].Required)
		require.Equal(t, enum.CheckStatusRunning, info.Checks[0].Check.Status)

		require.Equal(t, "lint", info.Checks[1].Check.Identifier)
		require.True(t, info.Checks[1].Required)
		require.Equal(t, enum.CheckStatusPending, info.Checks[1].Check.Status)
		require.Equal(t, int64(5), info.Checks[1].Check.RepoID)
		require.JSONEq(t, "{}", string(info.Checks[1].Check.Metadata))

		require.Equal(t, "test", info.Checks[2].Check.Identifier)
		require.False(t, info.Checks[2].Required)
	})

	t.Run("an entry without a checks commit only lists the required checks as pending", func(t *testing.T) {
		svc := &Service{
			mergeQueueEntryStore: infoEntryStore{entries: []*types.MergeQueueEntry{first}},
			checkStore:           infoCheckStore{err: errors.New("must not be called")},
		}

		info, err := svc.BuildMergeQueueInfo(t.Context(), repo, first, setup)
		require.NoError(t, err)
		require.Equal(t, 0, info.PullRequestsAhead)
		require.Len(t, info.Checks, 2)
		for _, c := range info.Checks {
			require.True(t, c.Required)
			require.Equal(t, enum.CheckStatusPending, c.Check.Status)
		}
	})

	t.Run("entry that is not in the queue", func(t *testing.T) {
		svc := &Service{mergeQueueEntryStore: infoEntryStore{entries: []*types.MergeQueueEntry{first}}}
		_, err := svc.BuildMergeQueueInfo(t.Context(), repo, second, setup)
		var userErr *usererror.Error
		require.ErrorAs(t, err, &userErr)
		require.Contains(t, userErr.Message, "not in merge queue")
	})

	t.Run("store errors", func(t *testing.T) {
		boom := errors.New("boom")
		svc := &Service{mergeQueueEntryStore: infoEntryStore{err: boom}}
		_, err := svc.BuildMergeQueueInfo(t.Context(), repo, first, setup)
		require.ErrorIs(t, err, boom)

		svc = &Service{
			mergeQueueEntryStore: infoEntryStore{entries: []*types.MergeQueueEntry{second}},
			checkStore:           infoCheckStore{err: boom},
		}
		_, err = svc.BuildMergeQueueInfo(t.Context(), repo, second, setup)
		require.ErrorIs(t, err, boom)
	})
}

func TestListMergeQueue(t *testing.T) {
	repo := &types.RepositoryCore{ID: 5}
	setup := protection.MergeQueueSetup{RequiredChecks: []string{"build"}}
	entries := []*types.MergeQueueEntry{infoEntry(10, sha.None), infoEntry(11, sha.None)}
	prs := map[int64]*types.PullReq{10: {ID: 10, Number: 1}, 11: {ID: 11, Number: 2}}

	newService := func() *Service {
		return &Service{
			mergeQueueStore:      infoQueueStore{queue: &types.MergeQueue{ID: 1}},
			mergeQueueEntryStore: infoEntryStore{entries: entries},
			pullreqStore:         infoPullReqStore{prs: prs},
			checkStore:           infoCheckStore{},
		}
	}

	t.Run("lists entries in queue order with their pull requests", func(t *testing.T) {
		items, err := newService().ListMergeQueue(t.Context(), repo, "main", setup)
		require.NoError(t, err)
		require.Len(t, items, 2)
		for i, item := range items {
			require.Equal(t, i, item.MergeQueueInfo.PullRequestsAhead)
			require.Equal(t, entries[i].PullReqID, item.PullReq.ID)
		}
	})

	t.Run("no merge queue yields an empty list", func(t *testing.T) {
		svc := newService()
		svc.mergeQueueStore = infoQueueStore{err: gitness_store.ErrResourceNotFound}
		items, err := svc.ListMergeQueue(t.Context(), repo, "main", setup)
		require.NoError(t, err)
		require.NotNil(t, items)
		require.Empty(t, items)
	})

	t.Run("errors", func(t *testing.T) {
		boom := errors.New("boom")

		svc := newService()
		svc.mergeQueueStore = infoQueueStore{err: boom}
		_, err := svc.ListMergeQueue(t.Context(), repo, "main", setup)
		require.ErrorIs(t, err, boom)

		svc = newService()
		svc.mergeQueueEntryStore = infoEntryStore{err: boom}
		_, err = svc.ListMergeQueue(t.Context(), repo, "main", setup)
		require.ErrorIs(t, err, boom)

		svc = newService()
		svc.pullreqStore = infoPullReqStore{prs: map[int64]*types.PullReq{10: prs[10]}}
		_, err = svc.ListMergeQueue(t.Context(), repo, "main", setup)
		require.Error(t, err)
		require.Contains(t, err.Error(), "11")

		svc = newService()
		svc.mergeQueueEntryStore = infoEntryStore{entries: []*types.MergeQueueEntry{infoEntry(10, sha.Must(checksSHA))}}
		svc.checkStore = infoCheckStore{err: boom}
		_, err = svc.ListMergeQueue(t.Context(), repo, "main", setup)
		require.ErrorIs(t, err, boom)
	})
}
