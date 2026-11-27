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

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbTriggerStore struct {
	store.TriggerStore

	triggers []*types.Trigger
	err      error
}

func (f *svbTriggerStore) ListAllEnabled(context.Context, int64) ([]*types.Trigger, error) {
	return f.triggers, f.err
}

type svbPipelineStore struct {
	store.PipelineStore

	pipelines map[int64]*types.Pipeline
}

func (f *svbPipelineStore) Find(_ context.Context, id int64) (*types.Pipeline, error) {
	p, ok := f.pipelines[id]
	if !ok {
		return nil, errSvbBoom
	}
	return p, nil
}

type svbTriggerer struct {
	fired []*types.Pipeline
	hooks []*triggerer.Hook
	errOn int64
}

func (f *svbTriggerer) Trigger(_ context.Context, p *types.Pipeline, h *triggerer.Hook) (*types.Execution, error) {
	f.fired = append(f.fired, p)
	f.hooks = append(f.hooks, h)
	if p.ID == f.errOn {
		return nil, errSvbBoom
	}
	return &types.Execution{}, nil
}

type svbPullReqStore struct {
	store.PullReqStore

	pr  *types.PullReq
	err error
}

func (f *svbPullReqStore) Find(context.Context, int64) (*types.PullReq, error) { return f.pr, f.err }

type svbCommitSvc struct {
	commit *types.Commit
	err    error
}

func (f svbCommitSvc) FindRef(context.Context, *types.RepositoryCore, string) (*types.Commit, error) {
	return f.commit, f.err
}

func (f svbCommitSvc) FindCommit(context.Context, *types.RepositoryCore, string) (*types.Commit, error) {
	return f.commit, f.err
}

type svbRepoIDCache struct {
	store.RepoIDCache

	err error
}

func (f *svbRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.RepositoryCore{ID: id}, nil
}

func svbRepoFinder(err error) refcache.RepoFinder {
	return refcache.NewRepoFinder(nil, nil, &svbRepoIDCache{err: err}, nil, cache.Evictor[*types.RepositoryCore]{})
}

func TestSvbConfigPrepare(t *testing.T) {
	var nilCfg *Config
	require.Error(t, nilCfg.Prepare())
	require.Error(t, (&Config{Concurrency: 1}).Prepare())
	require.Error(t, (&Config{EventReaderName: "r"}).Prepare())
	require.Error(t, (&Config{EventReaderName: "r", Concurrency: 1, MaxRetries: -1}).Prepare())
	require.NoError(t, (&Config{EventReaderName: "r", Concurrency: 1}).Prepare())
}

func TestSvbExtractBranch(t *testing.T) {
	require.Equal(t, "main", ExtractBranch("refs/heads/main"))
	require.Equal(t, "feature/x", ExtractBranch("refs/heads/feature/x"))
	require.Equal(t, "refs/tags/v1", ExtractBranch("refs/tags/v1"))
}

func TestSvbTrigger(t *testing.T) {
	ctx := context.Background()
	hook := &triggerer.Hook{Ref: "refs/heads/main"}

	newSvc := func(ts *svbTriggerStore, ts2 *svbTriggerer) *Service {
		return &Service{
			triggerStore: ts,
			pipelineStore: &svbPipelineStore{pipelines: map[int64]*types.Pipeline{
				1: {ID: 1},
				2: {ID: 2, Disabled: true},
				3: {ID: 3},
			}},
			triggerSvc: ts2,
		}
	}

	t.Run("fires matching triggers of enabled pipelines", func(t *testing.T) {
		tr := &svbTriggerer{}
		svc := newSvc(&svbTriggerStore{triggers: []*types.Trigger{
			{PipelineID: 1, Actions: []enum.TriggerAction{enum.TriggerActionBranchCreated}},
			{PipelineID: 1, Actions: []enum.TriggerAction{enum.TriggerActionTagCreated}},
			{PipelineID: 2, Actions: []enum.TriggerAction{enum.TriggerActionBranchCreated}},
			{PipelineID: 3, Actions: []enum.TriggerAction{
				enum.TriggerActionTagCreated, enum.TriggerActionBranchCreated,
			}},
		}}, tr)

		require.NoError(t, svc.trigger(ctx, 1, enum.TriggerActionBranchCreated, hook))
		require.Len(t, tr.fired, 2)
		require.Equal(t, int64(1), tr.fired[0].ID)
		require.Equal(t, int64(3), tr.fired[1].ID)
		require.Same(t, hook, tr.hooks[0])
	})

	t.Run("collects errors and keeps going", func(t *testing.T) {
		tr := &svbTriggerer{errOn: 1}
		svc := newSvc(&svbTriggerStore{triggers: []*types.Trigger{
			{PipelineID: 1, Actions: []enum.TriggerAction{enum.TriggerActionBranchCreated}},
			{PipelineID: 99, Actions: []enum.TriggerAction{enum.TriggerActionBranchCreated}},
			{PipelineID: 3, Actions: []enum.TriggerAction{enum.TriggerActionBranchCreated}},
		}}, tr)

		err := svc.trigger(ctx, 1, enum.TriggerActionBranchCreated, hook)
		require.ErrorIs(t, err, errSvbBoom)
		require.Len(t, tr.fired, 2, "pipeline lookups that fail are skipped, others still fire")
	})

	t.Run("list failure", func(t *testing.T) {
		svc := newSvc(&svbTriggerStore{err: errSvbBoom}, &svbTriggerer{})
		require.ErrorIs(t, svc.trigger(ctx, 1, enum.TriggerActionBranchCreated, hook), errSvbBoom)
	})
}

func TestSvbAugmentPullReqInfo(t *testing.T) {
	ctx := context.Background()
	pr := &types.PullReq{
		Number: 12, Title: "title", Description: "desc", Created: 1234, MergeBaseSHA: "base",
		TargetBranch: "main", SourceBranch: "feature",
		Author: types.PrincipalInfo{UID: "login", DisplayName: "Name", Email: "a@b.c"},
	}

	hook := &triggerer.Hook{}
	require.NoError(t, (&Service{pullReqStore: &svbPullReqStore{pr: pr}}).augmentPullReqInfo(ctx, hook, 1))
	require.Equal(t, "title", hook.Title)
	require.Equal(t, int64(1234), hook.Timestamp)
	require.Equal(t, "login", hook.AuthorLogin)
	require.Equal(t, "Name", hook.AuthorName)
	require.Equal(t, "a@b.c", hook.AuthorEmail)
	require.Equal(t, "desc", hook.Message)
	require.Equal(t, "base", hook.Before)
	require.Equal(t, "main", hook.Target)
	require.Equal(t, "feature", hook.Source)
	require.Equal(t, "refs/pullreq/12/head", hook.Ref)

	err := (&Service{pullReqStore: &svbPullReqStore{err: errSvbBoom}}).augmentPullReqInfo(ctx, hook, 1)
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbAugmentCommitInfo(t *testing.T) {
	ctx := context.Background()
	when := time.UnixMilli(5000)
	commit := &types.Commit{
		Title: "t", Message: "m",
		Author:    types.Signature{Identity: types.Identity{Name: "N", Email: "n@x.io"}},
		Committer: types.Signature{When: when},
	}

	hook := &triggerer.Hook{}
	svc := &Service{repoFinder: svbRepoFinder(nil), commitSvc: svbCommitSvc{commit: commit}}
	require.NoError(t, svc.augmentCommitInfo(ctx, hook, 1, "sha"))
	require.Equal(t, "N", hook.AuthorName)
	require.Equal(t, "N", hook.AuthorLogin)
	require.Equal(t, "n@x.io", hook.AuthorEmail)
	require.Equal(t, "t", hook.Title)
	require.Equal(t, "m", hook.Message)
	require.Equal(t, int64(5000), hook.Timestamp)

	svc = &Service{repoFinder: svbRepoFinder(errSvbBoom), commitSvc: svbCommitSvc{commit: commit}}
	require.ErrorIs(t, svc.augmentCommitInfo(ctx, hook, 1, "sha"), errSvbBoom)

	svc = &Service{repoFinder: svbRepoFinder(nil), commitSvc: svbCommitSvc{err: errSvbBoom}}
	require.ErrorContains(t, svc.augmentCommitInfo(ctx, hook, 1, "sha"), "could not find commit info")
}

type svbConsumer struct {
	registered int
}

func (c *svbConsumer) Register(string, stream.HandlerFunc, ...stream.HandlerOption) error {
	c.registered++
	return nil
}

func (c *svbConsumer) Configure(...stream.ConsumerOption) {}

func (c *svbConsumer) Start(context.Context) error { return nil }

func (c *svbConsumer) Errors() <-chan error { return make(chan error) }

func (c *svbConsumer) Infos() <-chan string { return make(chan string) }

type svbProducer struct{}

func (svbProducer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

func svbSystem(t *testing.T, consumer *svbConsumer, failAfter int) *events.System {
	t.Helper()
	calls := 0
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) {
			calls++
			if calls > failAfter {
				return nil, errSvbBoom
			}
			return consumer, nil
		},
		svbProducer{},
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	return system
}

func TestSvbNew(t *testing.T) {
	ctx := context.Background()
	cfg := Config{EventReaderName: "reader", Concurrency: 1}

	build := func(t *testing.T, system *events.System, cfg Config) (*Service, error) {
		t.Helper()
		gitFactory, err := gitevents.NewReaderFactory(system)
		require.NoError(t, err)
		prFactory, err := pullreqevents.NewReaderFactory(system)
		require.NoError(t, err)
		return New(ctx, cfg, nil, nil, refcache.RepoFinder{}, nil, nil, nil, gitFactory, prFactory)
	}

	t.Run("registers handlers", func(t *testing.T) {
		consumer := &svbConsumer{}
		svc, err := build(t, svbSystem(t, consumer, 2), cfg)
		require.NoError(t, err)
		require.NotNil(t, svc)
		require.Equal(t, 4+5, consumer.registered)
	})

	t.Run("invalid config", func(t *testing.T) {
		_, err := build(t, svbSystem(t, &svbConsumer{}, 2), Config{})
		require.Error(t, err)
	})

	t.Run("git reader failure", func(t *testing.T) {
		_, err := build(t, svbSystem(t, &svbConsumer{}, 0), cfg)
		require.ErrorContains(t, err, "git events reader")
	})

	t.Run("pull request reader failure", func(t *testing.T) {
		_, err := build(t, svbSystem(t, &svbConsumer{}, 1), cfg)
		require.ErrorContains(t, err, "pr events reader")
	})
}
