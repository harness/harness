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

package trigger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

const sv5SystemPrincipalID = 77

type sv5PrincipalStore struct {
	store.PrincipalStore
}

func (sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: sv5SystemPrincipalID, UID: "sys", Admin: true}, nil
}

// sv5Booted is set once at package initialization: the system service principal is package-level state.
var sv5Booted = sv5InitBootstrap()

func sv5InitBootstrap() bool {
	cfg := &types.Config{}
	cfg.Principal.System.UID = "sys"
	ctrl := service.NewController(check.PrincipalUIDDefault, nil, sv5PrincipalStore{})
	if err := bootstrap.SystemService(context.Background(), cfg, ctrl); err != nil {
		panic(err)
	}
	return true
}

func sv5Bootstrap(t *testing.T) {
	t.Helper()
	require.True(t, sv5Booted)
}

type sv5TriggerStore struct {
	store.TriggerStore

	triggers []*types.Trigger
	err      error
}

func (f *sv5TriggerStore) ListAllEnabled(context.Context, int64) ([]*types.Trigger, error) {
	return f.triggers, f.err
}

type sv5PipelineStore struct{ store.PipelineStore }

func (sv5PipelineStore) Find(_ context.Context, id int64) (*types.Pipeline, error) {
	return &types.Pipeline{ID: id}, nil
}

type sv5Triggerer struct {
	hooks []*triggerer.Hook
}

func (f *sv5Triggerer) Trigger(_ context.Context, _ *types.Pipeline, h *triggerer.Hook) (*types.Execution, error) {
	f.hooks = append(f.hooks, h)
	return &types.Execution{}, nil
}

type sv5PullReqStore struct {
	store.PullReqStore

	err error
}

func (f *sv5PullReqStore) Find(context.Context, int64) (*types.PullReq, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.PullReq{
		Number:       5,
		Title:        "PR title",
		Description:  "PR body",
		MergeBaseSHA: "base",
		SourceBranch: "feature",
		TargetBranch: "main",
		Created:      1234,
		Author:       types.PrincipalInfo{UID: "alice", DisplayName: "Alice", Email: "alice@example.com"},
	}, nil
}

type sv5CommitSvc struct{ err error }

func (f sv5CommitSvc) FindRef(context.Context, *types.RepositoryCore, string) (*types.Commit, error) {
	return nil, f.err
}

func (f sv5CommitSvc) FindCommit(context.Context, *types.RepositoryCore, string) (*types.Commit, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.Commit{
		Title:     "commit title",
		Message:   "commit message",
		Author:    types.Signature{Identity: types.Identity{Name: "Bob", Email: "bob@example.com"}},
		Committer: types.Signature{When: time.UnixMilli(4321)},
	}, nil
}

type sv5RepoIDCache struct{ store.RepoIDCache }

func (sv5RepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	return &types.RepositoryCore{ID: id}, nil
}

func sv5Service(action enum.TriggerAction, prErr, commitErr error) (*Service, *sv5Triggerer) {
	trig := &sv5Triggerer{}
	finder := refcache.NewRepoFinder(nil, nil, sv5RepoIDCache{}, nil, cache.Evictor[*types.RepositoryCore]{})
	return &Service{
		triggerStore:  &sv5TriggerStore{triggers: []*types.Trigger{{PipelineID: 1, Actions: []enum.TriggerAction{action}}}},
		pullReqStore:  &sv5PullReqStore{err: prErr},
		repoFinder:    finder,
		pipelineStore: sv5PipelineStore{},
		triggerSvc:    trig,
		commitSvc:     sv5CommitSvc{err: commitErr},
	}, trig
}

func sv5PRBase() pullreqevents.Base {
	return pullreqevents.Base{PullReqID: 1, TargetRepoID: 2}
}

func TestSv5PullReqHandlers(t *testing.T) {
	sv5Bootstrap(t)
	ctx := context.Background()
	base := sv5PRBase()

	type tc struct {
		name   string
		action enum.TriggerAction
		after  string
		run    func(s *Service) error
	}
	tests := []tc{
		{"created", enum.TriggerActionPullReqCreated, "src1", func(s *Service) error {
			return s.handleEventPullReqCreated(ctx, &events.Event[*pullreqevents.CreatedPayload]{
				Payload: &pullreqevents.CreatedPayload{Base: base, SourceSHA: "src1"}})
		}},
		{"reopened", enum.TriggerActionPullReqReopened, "src2", func(s *Service) error {
			return s.handleEventPullReqReopened(ctx, &events.Event[*pullreqevents.ReopenedPayload]{
				Payload: &pullreqevents.ReopenedPayload{Base: base, SourceSHA: "src2"}})
		}},
		{"branch updated", enum.TriggerActionPullReqBranchUpdated, "new3", func(s *Service) error {
			return s.handleEventPullReqBranchUpdated(ctx, &events.Event[*pullreqevents.BranchUpdatedPayload]{
				Payload: &pullreqevents.BranchUpdatedPayload{Base: base, NewSHA: "new3"}})
		}},
		{"closed", enum.TriggerActionPullReqClosed, "src4", func(s *Service) error {
			return s.handleEventPullReqClosed(ctx, &events.Event[*pullreqevents.ClosedPayload]{
				Payload: &pullreqevents.ClosedPayload{Base: base, SourceSHA: "src4"}})
		}},
		{"merged", enum.TriggerActionPullReqMerged, "src5", func(s *Service) error {
			return s.handleEventPullReqMerged(ctx, &events.Event[*pullreqevents.MergedPayload]{
				Payload: &pullreqevents.MergedPayload{Base: base, SourceSHA: "src5"}})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, trig := sv5Service(tt.action, nil, nil)
			require.NoError(t, tt.run(svc))
			require.Len(t, trig.hooks, 1)
			h := trig.hooks[0]
			require.Equal(t, enum.TriggerHook, h.Trigger)
			require.Equal(t, tt.action, h.Action)
			require.Equal(t, int64(sv5SystemPrincipalID), h.TriggeredBy)
			require.Equal(t, tt.after, h.After)
			require.Equal(t, "PR title", h.Title)
			require.Equal(t, "alice", h.AuthorLogin)
			require.Equal(t, "refs/pullreq/5/head", h.Ref)
			require.Equal(t, "main", h.Target)
			require.Equal(t, "feature", h.Source)

			failing, _ := sv5Service(tt.action, errSv5Boom, nil)
			require.ErrorContains(t, tt.run(failing), "could not augment pull request info")
		})
	}
}

func TestSv5GitHandlers(t *testing.T) {
	sv5Bootstrap(t)
	ctx := context.Background()

	type tc struct {
		name   string
		action enum.TriggerAction
		run    func(s *Service) error
		check  func(t *testing.T, h *triggerer.Hook)
	}
	tests := []tc{
		{"branch created", enum.TriggerActionBranchCreated, func(s *Service) error {
			return s.handleEventBranchCreated(ctx, &events.Event[*gitevents.BranchCreatedPayload]{
				Payload: &gitevents.BranchCreatedPayload{RepoID: 2, Ref: "refs/heads/dev", SHA: "s1"}})
		}, func(t *testing.T, h *triggerer.Hook) {
			require.Equal(t, "dev", h.Source)
			require.Equal(t, "dev", h.Target)
			require.Equal(t, "s1", h.After)
		}},
		{"branch updated", enum.TriggerActionBranchUpdated, func(s *Service) error {
			return s.handleEventBranchUpdated(ctx, &events.Event[*gitevents.BranchUpdatedPayload]{
				Payload: &gitevents.BranchUpdatedPayload{RepoID: 2, Ref: "refs/heads/dev", OldSHA: "o", NewSHA: "n"}})
		}, func(t *testing.T, h *triggerer.Hook) {
			require.Equal(t, "o", h.Before)
			require.Equal(t, "n", h.After)
			require.Equal(t, "dev", h.Target)
		}},
		{"tag created", enum.TriggerActionTagCreated, func(s *Service) error {
			return s.handleEventTagCreated(ctx, &events.Event[*gitevents.TagCreatedPayload]{
				Payload: &gitevents.TagCreatedPayload{RepoID: 2, Ref: "refs/tags/v1", SHA: "t1"}})
		}, func(t *testing.T, h *triggerer.Hook) {
			require.Equal(t, "refs/tags/v1", h.Ref)
			require.Equal(t, "t1", h.Before)
			require.Equal(t, "t1", h.After)
		}},
		{"tag updated", enum.TriggerActionTagUpdated, func(s *Service) error {
			return s.handleEventTagUpdated(ctx, &events.Event[*gitevents.TagUpdatedPayload]{
				Payload: &gitevents.TagUpdatedPayload{RepoID: 2, Ref: "refs/tags/v1", OldSHA: "a", NewSHA: "b"}})
		}, func(t *testing.T, h *triggerer.Hook) {
			require.Equal(t, "a", h.Before)
			require.Equal(t, "b", h.After)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, trig := sv5Service(tt.action, nil, nil)
			require.NoError(t, tt.run(svc))
			require.Len(t, trig.hooks, 1)
			h := trig.hooks[0]
			require.Equal(t, tt.action, h.Action)
			require.Equal(t, int64(sv5SystemPrincipalID), h.TriggeredBy)
			require.Equal(t, "Bob", h.AuthorName)
			require.Equal(t, "commit title", h.Title)
			require.Equal(t, int64(4321), h.Timestamp)
			tt.check(t, h)

			failing, _ := sv5Service(tt.action, nil, errSv5Boom)
			require.ErrorContains(t, tt.run(failing), "could not augment commit info")
		})
	}
}
