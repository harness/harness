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

package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5PrincipalStore struct{ store.PrincipalStore }

func (sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: 77, UID: "sys", DisplayName: "System", Email: "sys@example.com", Admin: true}, nil
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

type sv5RepoStore struct {
	store.RepoStore

	repo *types.Repository
	err  error
}

func (s *sv5RepoStore) Find(context.Context, int64) (*types.Repository, error) {
	return s.repo, s.err
}

type sv5Git struct {
	git.Interface

	err    error
	params *git.UpdateDefaultBranchParams
}

func (g *sv5Git) UpdateDefaultBranch(_ context.Context, p *git.UpdateDefaultBranchParams) error {
	g.params = p
	return g.err
}

type sv5URLProvider struct{ gitnessurl.Provider }

func (sv5URLProvider) GetInternalAPIURL(context.Context) string { return "http://internal.test" }

func sv5Locker() *locker.Locker {
	return locker.NewLocker(lock.NewInMemory(lock.Config{
		App: "test", Expiry: 10 * time.Second, Tries: 2, RetryDelay: 10 * time.Millisecond,
		DriftFactor: 0.01, TimeoutFactor: 0.25,
	}))
}

func TestSv5HandleUpdateDefaultBranch(t *testing.T) {
	require.True(t, sv5Booted)
	ctx := context.Background()
	event := &events.Event[*repoevents.DefaultBranchUpdatedPayload]{
		Payload: &repoevents.DefaultBranchUpdatedPayload{
			Base: repoevents.Base{RepoID: 5}, OldName: "main", NewName: "trunk",
		},
	}

	t.Run("updates the git default branch from the stored repository", func(t *testing.T) {
		g := &sv5Git{}
		svc := &Service{
			repoStore:   &sv5RepoStore{repo: &types.Repository{ID: 5, GitUID: "git-uid", DefaultBranch: "trunk"}},
			urlProvider: sv5URLProvider{},
			git:         g,
			locker:      sv5Locker(),
		}

		require.NoError(t, svc.handleUpdateDefaultBranch(ctx, event))
		require.NotNil(t, g.params)
		require.Equal(t, "trunk", g.params.BranchName)
		require.Equal(t, "git-uid", g.params.RepoUID)
		require.Equal(t, "System", g.params.Actor.Name)
		require.NotEmpty(t, g.params.EnvVars)
	})

	t.Run("repository lookup failure", func(t *testing.T) {
		svc := &Service{repoStore: &sv5RepoStore{err: errSv5Boom}, locker: sv5Locker()}
		err := svc.handleUpdateDefaultBranch(ctx, event)
		require.ErrorIs(t, err, errSv5Boom)
		require.ErrorContains(t, err, "find the repo")
	})

	t.Run("git failure", func(t *testing.T) {
		svc := &Service{
			repoStore:   &sv5RepoStore{repo: &types.Repository{ID: 5, DefaultBranch: "trunk"}},
			urlProvider: sv5URLProvider{},
			git:         &sv5Git{err: errSv5Boom},
			locker:      sv5Locker(),
		}
		require.ErrorContains(t, svc.handleUpdateDefaultBranch(ctx, event), "update the repo default branch to trunk")
	})

	t.Run("lock is held by someone else", func(t *testing.T) {
		l := sv5Locker()
		unlock, err := l.LockDefaultBranch(ctx, 5, "trunk", time.Minute)
		require.NoError(t, err)
		defer unlock()

		svc := &Service{repoStore: &sv5RepoStore{}, locker: l}
		require.ErrorContains(t, svc.handleUpdateDefaultBranch(ctx, event), "failed to lock repo")
	})
}

type sv5Consumer struct{ registered int }

func (c *sv5Consumer) Register(string, stream.HandlerFunc, ...stream.HandlerOption) error {
	c.registered++
	return nil
}

func (c *sv5Consumer) Configure(...stream.ConsumerOption) {}

func (c *sv5Consumer) Start(context.Context) error { return nil }

func (c *sv5Consumer) Errors() <-chan error { return make(chan error) }

func (c *sv5Consumer) Infos() <-chan string { return make(chan string) }

type sv5Producer struct{}

func (sv5Producer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

func TestSv5NewService(t *testing.T) {
	ctx := context.Background()
	cfg := &types.Config{InstanceID: "instance"}

	build := func(t *testing.T, okLaunches int) (*Service, *sv5Consumer, error) {
		t.Helper()
		consumer := &sv5Consumer{}
		calls := 0
		system, err := events.NewSystem(
			func(string, string) (events.StreamConsumer, error) {
				calls++
				if calls > okLaunches {
					return nil, errSv5Boom
				}
				return consumer, nil
			},
			sv5Producer{},
			events.NewNoopCollector(),
			false,
		)
		require.NoError(t, err)
		factory, err := repoevents.NewReaderFactory(system)
		require.NoError(t, err)
		svc, err := NewService(ctx, cfg, nil, factory, &sv5RepoStore{}, sv5URLProvider{}, &sv5Git{}, sv5Locker())
		return svc, consumer, err
	}

	svc, consumer, err := build(t, 2)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, 2, consumer.registered)

	_, _, err = build(t, 0)
	require.ErrorIs(t, err, errSv5Boom)
	require.ErrorContains(t, err, "repo git group")

	_, _, err = build(t, 1)
	require.ErrorIs(t, err, errSv5Boom)
	require.ErrorContains(t, err, "last git push")
}

func TestSv5NoopListener(t *testing.T) {
	var l Listener = noopListener{}
	require.NoError(t, l.AfterSizeUpdate(context.Background(), 1, 2))
	l.AfterSizeCalculation(context.Background(), 2)
}

type sv5JobStore struct {
	job.Store

	err error
	uid string
}

func (s *sv5JobStore) Upsert(_ context.Context, j *job.Job) error {
	s.uid = j.UID
	return s.err
}

func TestSv5SizeCalculatorRegister(t *testing.T) {
	ctx := context.Background()

	newCalc := func(t *testing.T, st *sv5JobStore, enabled bool, cron string) *SizeCalculator {
		t.Helper()
		scheduler, err := job.NewScheduler(st, nil, nil, nil, "inst", 1, time.Hour)
		require.NoError(t, err)
		return &SizeCalculator{enabled: enabled, cron: cron, maxDur: time.Minute, scheduler: scheduler}
	}

	disabled := &sv5JobStore{}
	require.NoError(t, newCalc(t, disabled, false, "0 0 * * *").Register(ctx))
	require.Empty(t, disabled.uid)

	enabled := &sv5JobStore{}
	require.NoError(t, newCalc(t, enabled, true, "0 0 * * *").Register(ctx))
	require.Equal(t, jobType, enabled.uid)

	require.ErrorIs(t, newCalc(t, &sv5JobStore{err: errSv5Boom}, true, "0 0 * * *").Register(ctx), errSv5Boom)
	require.ErrorContains(t, newCalc(t, &sv5JobStore{}, true, "bad cron").Register(ctx), "recurring job for calculator")
}

type sv5SpaceStore struct {
	store.SpaceStore

	spaces []types.SpaceStorage
	err    error
}

func (s sv5SpaceStore) GetRootSpacesSize(context.Context) ([]types.SpaceStorage, error) {
	return s.spaces, s.err
}

type sv5UsageStore struct {
	store.UsageMetricStore

	err     error
	metrics []*types.UsageMetric
}

func (s *sv5UsageStore) UpsertStorage(_ context.Context, in []*types.UsageMetric) error {
	s.metrics = in
	return s.err
}

func TestSv5SizeCalculatorSendMetric(t *testing.T) {
	ctx := context.Background()
	spaces := []types.SpaceStorage{{ID: 1, Identifier: "root", Size: 10, LFSSize: 4}}

	t.Run("midnight cron reports the previous day", func(t *testing.T) {
		usage := &sv5UsageStore{}
		calc := &SizeCalculator{cron: "0 0 * * *", spaceStore: sv5SpaceStore{spaces: spaces}, usageMetricStore: usage}

		require.NoError(t, calc.sendMetric(ctx))
		require.Len(t, usage.metrics, 1)
		require.Equal(t, int64(10), usage.metrics[0].StorageTotal)
		require.Equal(t, int64(4), usage.metrics[0].LFSStorageTotal)
		require.Equal(t, "root", usage.metrics[0].RootSpaceIdentifier)
		require.Less(t, time.Since(usage.metrics[0].Date), 25*time.Hour)
		require.Greater(t, time.Since(usage.metrics[0].Date), 23*time.Hour)
	})

	t.Run("other cron reports today", func(t *testing.T) {
		usage := &sv5UsageStore{}
		calc := &SizeCalculator{cron: "30 3 * * *", spaceStore: sv5SpaceStore{spaces: spaces}, usageMetricStore: usage}

		require.NoError(t, calc.sendMetric(ctx))
		require.Less(t, time.Since(usage.metrics[0].Date), time.Hour)
	})

	t.Run("space listing failure", func(t *testing.T) {
		calc := &SizeCalculator{spaceStore: sv5SpaceStore{err: errSv5Boom}}
		require.ErrorIs(t, calc.sendMetric(ctx), errSv5Boom)
	})

	t.Run("upsert failure is only logged", func(t *testing.T) {
		calc := &SizeCalculator{
			spaceStore: sv5SpaceStore{spaces: spaces}, usageMetricStore: &sv5UsageStore{err: errSv5Boom},
		}
		require.NoError(t, calc.sendMetric(ctx))
	})
}
