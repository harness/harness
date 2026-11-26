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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbTokenStore struct {
	store.TokenStore

	n      int64
	err    error
	before time.Time
	types  []enum.TokenType
}

func (f *svbTokenStore) DeleteExpiredBefore(
	_ context.Context, before time.Time, types []enum.TokenType,
) (int64, error) {
	f.before, f.types = before, types
	return f.n, f.err
}

type svbWebhookExecStore struct {
	store.WebhookExecutionStore

	n         int64
	err       error
	olderThan time.Time
}

func (f *svbWebhookExecStore) DeleteOld(_ context.Context, olderThan time.Time) (int64, error) {
	f.olderThan = olderThan
	return f.n, f.err
}

func svbValidConfig() Config {
	return Config{
		WebhookExecutionsRetentionTime:        time.Hour,
		DeletedRepositoriesRetentionTime:      time.Hour,
		DeletedRepositoriesCleanupCron:        "0 * * * *",
		DeletedRepositoriesCleanupMaxDuration: time.Minute,
	}
}

func TestSvbConfigPrepare(t *testing.T) {
	var nilCfg *Config
	require.Error(t, nilCfg.Prepare())

	valid := svbValidConfig()
	require.NoError(t, valid.Prepare())

	tests := []struct {
		name string
		mod  func(c *Config)
	}{
		{"webhook retention", func(c *Config) { c.WebhookExecutionsRetentionTime = 0 }},
		{"deleted repo retention", func(c *Config) { c.DeletedRepositoriesRetentionTime = -1 }},
		{"cron", func(c *Config) { c.DeletedRepositoriesCleanupCron = "" }},
		{"max duration", func(c *Config) { c.DeletedRepositoriesCleanupMaxDuration = 0 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := svbValidConfig()
			tc.mod(&cfg)
			require.Error(t, cfg.Prepare())

			_, err := NewService(cfg, nil, nil, nil, nil, nil, nil)
			require.ErrorContains(t, err, "cleanup config is invalid")
		})
	}
}

func TestSvbNewServiceAndRegisterHandlers(t *testing.T) {
	svc, err := NewService(svbValidConfig(), nil, job.NewExecutor(nil, nil), nil, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, svc)

	require.NoError(t, svc.registerJobHandlers())

	// Registering the same job types again must fail.
	err = svc.registerJobHandlers()
	require.ErrorContains(t, err, "webhook executions cleanup")

	err = svc.Register(context.Background())
	require.ErrorContains(t, err, "failed to register cleanup job handlers")
}

func TestSvbTokensCleanupJobHandle(t *testing.T) {
	ctx := context.Background()

	ts := &svbTokenStore{n: 3}
	got, err := newTokensCleanupJob(ts).Handle(ctx, "", nil)
	require.NoError(t, err)
	require.Equal(t, "deleted 3 tokens", got)
	require.Equal(t, []enum.TokenType{enum.TokenTypeSession}, ts.types)
	require.WithinDuration(t, time.Now().Add(-tokenRetentionTime), ts.before, time.Minute)

	ts = &svbTokenStore{}
	got, err = newTokensCleanupJob(ts).Handle(ctx, "", nil)
	require.NoError(t, err)
	require.Equal(t, "no expired tokens found", got)

	ts = &svbTokenStore{err: errSvbBoom}
	_, err = newTokensCleanupJob(ts).Handle(ctx, "", nil)
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbWebhookExecutionsCleanupJobHandle(t *testing.T) {
	ctx := context.Background()

	ws := &svbWebhookExecStore{n: 7}
	got, err := newWebhookExecutionsCleanupJob(2*time.Hour, ws).Handle(ctx, "", nil)
	require.NoError(t, err)
	require.Equal(t, "deleted 7 webhook executions", got)
	require.WithinDuration(t, time.Now().Add(-2*time.Hour), ws.olderThan, time.Minute)

	ws = &svbWebhookExecStore{}
	got, err = newWebhookExecutionsCleanupJob(time.Hour, ws).Handle(ctx, "", nil)
	require.NoError(t, err)
	require.Equal(t, "no old webhook executions found", got)

	ws = &svbWebhookExecStore{err: errSvbBoom}
	_, err = newWebhookExecutionsCleanupJob(time.Hour, ws).Handle(ctx, "", nil)
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbNewDeletedReposCleanupJob(t *testing.T) {
	j := newDeletedReposCleanupJob(time.Hour, nil, nil)
	require.Equal(t, time.Hour, j.retentionTime)
}
