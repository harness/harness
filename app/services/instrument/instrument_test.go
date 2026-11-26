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

package instrument

import (
	"context"
	"errors"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type recordingService struct {
	tracked []Event
	err     error
}

func (r *recordingService) Track(_ context.Context, e Event) error {
	r.tracked = append(r.tracked, e)
	return r.err
}

func (r *recordingService) Close(context.Context) error { return nil }

type fakePrincipalCache struct {
	store.PrincipalInfoCache

	info *types.PrincipalInfo
	err  error
}

func (f fakePrincipalCache) Get(context.Context, int64) (*types.PrincipalInfo, error) {
	return f.info, f.err
}

func TestEventType_String(t *testing.T) {
	require.Equal(t, "Create commit", EventTypeCreateCommit.String())
	require.Equal(t, "Merge pull request", EventTypeMergePullRequest.String())
}

func TestNoop(t *testing.T) {
	var svc Service = Noop{}
	require.NoError(t, svc.Track(t.Context(), Event{Type: EventTypeCreateBranch}))
	require.NoError(t, svc.Close(t.Context()))
}

func branchUpdated(ref string) *events.Event[*gitevents.BranchUpdatedPayload] {
	return &events.Event[*gitevents.BranchUpdatedPayload]{
		Payload: &gitevents.BranchUpdatedPayload{RepoID: 3, PrincipalID: 9, Ref: ref},
	}
}

func TestConsumer_InstrumentTrackOnBranchUpdate(t *testing.T) {
	repo := &types.Repository{ID: 3, Identifier: "my-repo", Path: "space/my-repo", DefaultBranch: "main"}
	principal := &types.PrincipalInfo{ID: 9, UID: "jane"}

	newConsumer := func(repoStore *mockstore.RepoStore, cache fakePrincipalCache, svc Service) Consumer {
		return Consumer{repoStore: repoStore, principalInfoCache: cache, instrumentation: svc}
	}

	t.Run("tracks a commit on the default branch", func(t *testing.T) {
		repoStore := &mockstore.RepoStore{}
		repoStore.On("Find", int64(3)).Return(repo, nil)
		svc := &recordingService{}

		err := newConsumer(repoStore, fakePrincipalCache{info: principal}, svc).
			instrumentTrackOnBranchUpdate(t.Context(), branchUpdated("main"))
		require.NoError(t, err)

		require.Len(t, svc.tracked, 1)
		got := svc.tracked[0]
		require.Equal(t, EventTypeCreateCommit, got.Type)
		require.Equal(t, principal, got.Principal)
		require.Equal(t, "space/my-repo", got.Path)
		require.Equal(t, map[Property]any{
			PropertyRepositoryID:    int64(3),
			PropertyRepositoryName:  "my-repo",
			PropertyIsDefaultBranch: true,
		}, got.Properties)
	})

	t.Run("flags other branches", func(t *testing.T) {
		repoStore := &mockstore.RepoStore{}
		repoStore.On("Find", int64(3)).Return(repo, nil)
		svc := &recordingService{}

		require.NoError(t, newConsumer(repoStore, fakePrincipalCache{info: principal}, svc).
			instrumentTrackOnBranchUpdate(t.Context(), branchUpdated("feature")))
		require.Equal(t, false, svc.tracked[0].Properties[PropertyIsDefaultBranch])
	})

	t.Run("instrumentation failures do not fail the event", func(t *testing.T) {
		repoStore := &mockstore.RepoStore{}
		repoStore.On("Find", int64(3)).Return(repo, nil)
		svc := &recordingService{err: errors.New("posthog down")}

		require.NoError(t, newConsumer(repoStore, fakePrincipalCache{info: principal}, svc).
			instrumentTrackOnBranchUpdate(t.Context(), branchUpdated("main")))
		require.Len(t, svc.tracked, 1)
	})

	t.Run("repo lookup failure is returned", func(t *testing.T) {
		repoStore := &mockstore.RepoStore{}
		repoStore.On("Find", int64(3)).Return(nil, errors.New("boom"))
		svc := &recordingService{}

		err := newConsumer(repoStore, fakePrincipalCache{info: principal}, svc).
			instrumentTrackOnBranchUpdate(t.Context(), branchUpdated("main"))
		require.Error(t, err)
		require.Empty(t, svc.tracked)
	})

	t.Run("principal lookup failure is returned", func(t *testing.T) {
		repoStore := &mockstore.RepoStore{}
		repoStore.On("Find", int64(3)).Return(repo, nil)
		svc := &recordingService{}

		err := newConsumer(repoStore, fakePrincipalCache{err: errors.New("boom")}, svc).
			instrumentTrackOnBranchUpdate(t.Context(), branchUpdated("main"))
		require.Error(t, err)
		require.Empty(t, svc.tracked)
	})
}
