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

package cleanup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5PrincipalStore struct{ store.PrincipalStore }

func (sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: 77, UID: "sys", Admin: true}, nil
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

	err   error
	calls int
	got   *types.RepoFilter
}

func (s *sv5RepoStore) ListAll(_ context.Context, filter *types.RepoFilter) ([]*types.Repository, error) {
	s.calls++
	s.got = filter
	return nil, s.err
}

func TestSv5DeletedReposHandle(t *testing.T) {
	require.True(t, sv5Booted)
	ctx := context.Background()

	t.Run("nothing to purge", func(t *testing.T) {
		repos := &sv5RepoStore{}
		j := newDeletedReposCleanupJob(time.Hour, repos, nil)

		before := time.Now().Add(-time.Hour).UnixMilli()
		res, err := j.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Equal(t, "no old deleted repositories found", res)
		require.Equal(t, 1, repos.calls)
		require.NotNil(t, repos.got.DeletedBeforeOrAt)
		require.GreaterOrEqual(t, *repos.got.DeletedBeforeOrAt, before)
		require.Equal(t, maxDeletedRepoRetrival, repos.got.Size)
	})

	t.Run("listing failure", func(t *testing.T) {
		j := newDeletedReposCleanupJob(time.Hour, &sv5RepoStore{err: errSv5Boom}, nil)
		_, err := j.Handle(ctx, "", nil)
		require.ErrorIs(t, err, errSv5Boom)
		require.ErrorContains(t, err, "list ready-to-delete repositories")
	})
}

type sv5JobStore struct {
	job.Store

	upserts int
	failAt  int
	uids    []string
}

func (s *sv5JobStore) Upsert(_ context.Context, j *job.Job) error {
	s.upserts++
	s.uids = append(s.uids, j.UID)
	if s.upserts == s.failAt {
		return errSv5Boom
	}
	return nil
}

func sv5Config() Config {
	return Config{
		WebhookExecutionsRetentionTime:        time.Hour,
		DeletedRepositoriesRetentionTime:      time.Hour,
		DeletedRepositoriesCleanupCron:        "0 0 * * *",
		DeletedRepositoriesCleanupMaxDuration: time.Minute,
	}
}

func sv5NewService(t *testing.T, st *sv5JobStore, cfg Config) *Service {
	t.Helper()
	scheduler, err := job.NewScheduler(st, nil, nil, nil, "inst", 1, time.Hour)
	require.NoError(t, err)
	svc, err := NewService(cfg, scheduler, job.NewExecutor(st, nil), nil, nil, &sv5RepoStore{}, nil)
	require.NoError(t, err)
	return svc
}

func TestSv5ServiceRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("schedules all recurring jobs", func(t *testing.T) {
		st := &sv5JobStore{}
		svc := sv5NewService(t, st, sv5Config())
		require.NoError(t, svc.Register(ctx))
		require.Equal(t,
			[]string{jobTypeWebhookExecutions, jobTypeTokens, jobTypeDeletedRepos}, st.uids)
	})

	for failAt, msg := range map[int]string{
		1: "webhook executions job",
		2: "token job",
		3: "deleted repo cleanup job",
	} {
		t.Run("schedule failure "+msg, func(t *testing.T) {
			svc := sv5NewService(t, &sv5JobStore{failAt: failAt}, sv5Config())
			err := svc.Register(ctx)
			require.ErrorIs(t, err, errSv5Boom)
			require.ErrorContains(t, err, msg)
		})
	}

	t.Run("invalid cron", func(t *testing.T) {
		cfg := sv5Config()
		cfg.DeletedRepositoriesCleanupCron = "not a cron"
		svc := sv5NewService(t, &sv5JobStore{}, cfg)
		require.ErrorContains(t, svc.Register(ctx), "deleted repo cleanup job")
	})

	t.Run("handlers are registered only once", func(t *testing.T) {
		svc := sv5NewService(t, &sv5JobStore{}, sv5Config())
		require.NoError(t, svc.registerJobHandlers())
		require.ErrorContains(t, svc.Register(ctx), "register cleanup job handlers")
	})
}

func TestSv5ConfigPrepare(t *testing.T) {
	var nilConfig *Config
	require.ErrorContains(t, nilConfig.Prepare(), "config is required")

	require.NoError(t, (&Config{
		WebhookExecutionsRetentionTime:        time.Hour,
		DeletedRepositoriesRetentionTime:      time.Hour,
		DeletedRepositoriesCleanupCron:        "0 0 * * *",
		DeletedRepositoriesCleanupMaxDuration: time.Minute,
	}).Prepare())

	tests := map[string]func(c *Config){
		"WebhookExecutionsRetentionTime":        func(c *Config) { c.WebhookExecutionsRetentionTime = 0 },
		"DeletedRepositoriesRetentionTime":      func(c *Config) { c.DeletedRepositoriesRetentionTime = 0 },
		"DeletedRepositoriesCleanupCron":        func(c *Config) { c.DeletedRepositoriesCleanupCron = "" },
		"DeletedRepositoriesCleanupMaxDuration": func(c *Config) { c.DeletedRepositoriesCleanupMaxDuration = 0 },
	}
	for field, mutate := range tests {
		t.Run(field, func(t *testing.T) {
			cfg := sv5Config()
			mutate(&cfg)
			require.ErrorContains(t, cfg.Prepare(), field)

			_, err := NewService(cfg, nil, nil, nil, nil, nil, nil)
			require.ErrorContains(t, err, "cleanup config is invalid")
		})
	}
}
