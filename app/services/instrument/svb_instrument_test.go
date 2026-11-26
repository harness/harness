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

package instrument

import (
	"context"
	"errors"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbService struct {
	events []Event
	err    error
}

func (s *svbService) Track(_ context.Context, e Event) error {
	s.events = append(s.events, e)
	return s.err
}

func (s *svbService) Close(context.Context) error { return nil }

type svbRepoStore struct {
	store.RepoStore

	repo     *types.Repository
	findErr  error
	counts   []types.RepositoryCount
	countErr error
}

func (f *svbRepoStore) Find(context.Context, int64) (*types.Repository, error) {
	return f.repo, f.findErr
}

func (f *svbRepoStore) CountByRootSpaces(context.Context) ([]types.RepositoryCount, error) {
	return f.counts, f.countErr
}

type svbPrincipalCache struct {
	store.PrincipalInfoCache

	info *types.PrincipalInfo
	err  error
}

func (f *svbPrincipalCache) Get(context.Context, int64) (*types.PrincipalInfo, error) {
	return f.info, f.err
}

func TestSvbEventTypeString(t *testing.T) {
	require.Equal(t, "Create commit", EventTypeCreateCommit.String())
	require.Equal(t, "Repository count", EventTypeRepositoryCount.String())
}

func TestSvbNoop(t *testing.T) {
	var s Service = Noop{}
	require.NoError(t, s.Track(context.Background(), Event{}))
	require.NoError(t, s.Close(context.Background()))
}

func TestSvbInstrumentTrackOnBranchUpdate(t *testing.T) {
	ctx := context.Background()
	repo := &types.Repository{ID: 4, Identifier: "r", Path: "a/r", DefaultBranch: "refs/heads/main"}
	info := &types.PrincipalInfo{ID: 9}
	ev := func(ref string) *events.Event[*gitevents.BranchUpdatedPayload] {
		return &events.Event[*gitevents.BranchUpdatedPayload]{
			Payload: &gitevents.BranchUpdatedPayload{RepoID: 4, PrincipalID: 9, Ref: ref},
		}
	}

	t.Run("tracks the commit", func(t *testing.T) {
		svc := &svbService{}
		c := Consumer{
			repoStore:          &svbRepoStore{repo: repo},
			principalInfoCache: &svbPrincipalCache{info: info},
			instrumentation:    svc,
		}
		require.NoError(t, c.instrumentTrackOnBranchUpdate(ctx, ev("refs/heads/main")))
		require.Len(t, svc.events, 1)
		got := svc.events[0]
		require.Equal(t, EventTypeCreateCommit, got.Type)
		require.Equal(t, "a/r", got.Path)
		require.Same(t, info, got.Principal)
		require.Equal(t, int64(4), got.Properties[PropertyRepositoryID])
		require.Equal(t, "r", got.Properties[PropertyRepositoryName])
		require.Equal(t, true, got.Properties[PropertyIsDefaultBranch])

		require.NoError(t, c.instrumentTrackOnBranchUpdate(ctx, ev("refs/heads/dev")))
		require.Equal(t, false, svc.events[1].Properties[PropertyIsDefaultBranch])
	})

	t.Run("tracking failures are tolerated", func(t *testing.T) {
		c := Consumer{
			repoStore:          &svbRepoStore{repo: repo},
			principalInfoCache: &svbPrincipalCache{info: info},
			instrumentation:    &svbService{err: errSvbBoom},
		}
		require.NoError(t, c.instrumentTrackOnBranchUpdate(ctx, ev("refs/heads/main")))
	})

	t.Run("lookup failures", func(t *testing.T) {
		c := Consumer{
			repoStore:          &svbRepoStore{findErr: errSvbBoom},
			principalInfoCache: &svbPrincipalCache{info: info},
			instrumentation:    &svbService{},
		}
		require.ErrorIs(t, c.instrumentTrackOnBranchUpdate(ctx, ev("x")), errSvbBoom)

		c = Consumer{
			repoStore:          &svbRepoStore{repo: repo},
			principalInfoCache: &svbPrincipalCache{err: errSvbBoom},
			instrumentation:    &svbService{},
		}
		require.ErrorIs(t, c.instrumentTrackOnBranchUpdate(ctx, ev("x")), errSvbBoom)
	})
}

type svbConsumer struct{ registered int }

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

func TestSvbNewConsumer(t *testing.T) {
	ctx := context.Background()
	build := func(consumer *svbConsumer, factoryErr error) (Consumer, error) {
		system, err := events.NewSystem(
			func(string, string) (events.StreamConsumer, error) {
				if factoryErr != nil {
					return nil, factoryErr
				}
				return consumer, nil
			},
			svbProducer{},
			events.NewNoopCollector(),
		)
		require.NoError(t, err)
		factory, err := gitevents.NewReaderFactory(system)
		require.NoError(t, err)
		return NewConsumer(ctx, &types.Config{InstanceID: "inst"}, factory, nil, nil, Noop{})
	}

	consumer := &svbConsumer{}
	_, err := build(consumer, nil)
	require.NoError(t, err)
	require.Equal(t, 1, consumer.registered)

	_, err = build(nil, errSvbBoom)
	require.ErrorContains(t, err, "failed to launch git consumer")
}

func TestSvbRepositoryCountHandle(t *testing.T) {
	ctx := context.Background()

	disabled := &RepositoryCount{enabled: false}
	_, err := disabled.Handle(ctx, "", nil)
	require.ErrorContains(t, err, "disabled")

	noStore := &RepositoryCount{enabled: true}
	_, err = noStore.Handle(ctx, "", nil)
	require.ErrorContains(t, err, "repository store not initialized")

	failing := &RepositoryCount{enabled: true, repoStore: &svbRepoStore{countErr: errSvbBoom}}
	_, err = failing.Handle(ctx, "", nil)
	require.ErrorIs(t, err, errSvbBoom)

	noSvc := &RepositoryCount{enabled: true, repoStore: &svbRepoStore{}}
	_, err = noSvc.Handle(ctx, "", nil)
	require.ErrorContains(t, err, "service not initialized")
}

func TestSvbNewRepositoryCount(t *testing.T) {
	ctx := context.Background()
	cfg := &types.Config{}
	cfg.Instrumentation.Enable = true

	// A nil scheduler is rejected after the job handler registration succeeded.
	executor := job.NewExecutor(nil, nil)
	_, err := NewRepositoryCount(ctx, cfg, Noop{}, nil, nil, executor)
	require.ErrorContains(t, err, "job scheduler is nil")

	// The job type is already registered by the first attempt.
	_, err = NewRepositoryCount(ctx, cfg, Noop{}, nil, nil, executor)
	require.ErrorContains(t, err, "failed to register instrumentation job")
}
