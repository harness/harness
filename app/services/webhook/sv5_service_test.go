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

package webhook

import (
	"context"
	"errors"
	"testing"

	checkevents "github.com/harness/gitness/app/events/check"
	gitevents "github.com/harness/gitness/app/events/git"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

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

func sv5System(t *testing.T, consumer *sv5Consumer, okLaunches int) *events.System {
	t.Helper()
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
	return system
}

func sv5NewService(t *testing.T, system *events.System, cfg Config) (*Service, error) {
	t.Helper()
	gitFactory, err := gitevents.NewReaderFactory(system)
	require.NoError(t, err)
	prFactory, err := pullreqevents.NewReaderFactory(system)
	require.NoError(t, err)
	mqFactory, err := mergequeueevents.NewReaderFactory(system)
	require.NoError(t, err)
	checkFactory, err := checkevents.NewReaderFactory(system)
	require.NoError(t, err)

	return NewService(
		context.Background(), cfg, nil, gitFactory, prFactory, mqFactory, checkFactory,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
}

func TestSv5ConfigPrepare(t *testing.T) {
	var nilConfig *Config
	require.ErrorContains(t, nilConfig.Prepare(), "config is required")

	tests := []struct {
		name string
		cfg  Config
		msg  string
	}{
		{"reader name", Config{UserAgentIdentity: "ua", Concurrency: 1}, "EventReaderName"},
		{"user agent", Config{EventReaderName: "r", Concurrency: 1}, "UserAgentIdentity"},
		{"concurrency", Config{EventReaderName: "r", UserAgentIdentity: "ua"}, "Concurrency"},
		{"retries", Config{EventReaderName: "r", UserAgentIdentity: "ua", Concurrency: 1, MaxRetries: -1}, "MaxRetries"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorContains(t, tc.cfg.Prepare(), tc.msg)
		})
	}

	cfg := Config{EventReaderName: "r", UserAgentIdentity: "ua", Concurrency: 1}
	require.NoError(t, cfg.Prepare())
	require.Equal(t, "ua", cfg.HeaderIdentity, "header identity falls back to the user agent identity")
}

func TestSv5NewService(t *testing.T) {
	cfg := Config{EventReaderName: "reader", UserAgentIdentity: "ua", Concurrency: 1}

	t.Run("registers all handlers", func(t *testing.T) {
		consumer := &sv5Consumer{}
		svc, err := sv5NewService(t, sv5System(t, consumer, 4), cfg)
		require.NoError(t, err)
		require.NotNil(t, svc)
		require.NotNil(t, svc.WebhookExecutor)
		// 6 git + 17 pull request + 1 check + 2 merge queue events.
		require.Equal(t, 6+17+1+2, consumer.registered)
	})

	t.Run("invalid config", func(t *testing.T) {
		_, err := sv5NewService(t, sv5System(t, &sv5Consumer{}, 4), Config{})
		require.ErrorContains(t, err, "webhook service Config is invalid")
	})

	for launched, msg := range map[int]string{
		0: "git event reader",
		1: "pr event reader",
		2: "check event reader",
		3: "merge queue event reader",
	} {
		t.Run("launch failure "+msg, func(t *testing.T) {
			_, err := sv5NewService(t, sv5System(t, &sv5Consumer{}, launched), cfg)
			require.ErrorIs(t, err, errSv5Boom)
			require.ErrorContains(t, err, msg)
		})
	}
}

func TestSv5NewWebhookExecutor(t *testing.T) {
	cfg := Config{UserAgentIdentity: "ua", HeaderIdentity: "hdr"}
	executor := NewWebhookExecutor(cfg, nil, nil, nil, nil, nil, nil, RepoTrigger)

	require.NotNil(t, executor.secureHTTPClient)
	require.NotNil(t, executor.insecureHTTPClient)
	require.NotNil(t, executor.secureHTTPClientInternal)
	require.NotNil(t, executor.insecureHTTPClientInternal)
	require.NotSame(t, executor.secureHTTPClient, executor.insecureHTTPClient)
	require.Equal(t, RepoTrigger, executor.source)
	require.Equal(t, "hdr", executor.config.HeaderIdentity)
}

func TestSv5AddrClassError(t *testing.T) {
	require.ErrorIs(t, addrClassError(netpolicy.ClassLoopback), errLoopbackNotAllowed)
	require.ErrorIs(t, addrClassError(netpolicy.ClassLinkLocal), errLinkLocalNotAllowed)
	require.ErrorIs(t, addrClassError(netpolicy.ClassPrivate), errPrivateNetworkNotAllowed)
	require.ErrorIs(t, addrClassError(netpolicy.ClassReserved), errPrivateNetworkNotAllowed)
	require.NoError(t, addrClassError(netpolicy.ClassPublic))
	require.NoError(t, addrClassError(netpolicy.Class(99)))
}

type sv5URLProvider struct{ gitnessurl.Provider }

func (sv5URLProvider) GenerateUIRefURL(_ context.Context, repoPath, ref string) string {
	return "https://ui/" + repoPath + "/commit/" + ref
}

func TestSv5CommitInfoFrom(t *testing.T) {
	commit := git.Commit{
		SHA:     sha.Must("1111111111111111111111111111111111111111"),
		Message: "msg",
		FileStats: []git.CommitFileStats{
			{Status: gitenum.FileDiffStatusAdded, Path: "added.txt"},
			{Status: gitenum.FileDiffStatusCopied, Path: "copied.txt"},
			{Status: gitenum.FileDiffStatusModified, Path: "modified.txt"},
			{Status: gitenum.FileDiffStatusDeleted, Path: "deleted.txt"},
			{Status: gitenum.FileDiffStatusRenamed, Path: "new.txt", OldPath: "old.txt"},
			{Status: gitenum.FileDiffStatusUndefined, Path: "ignored.txt"},
			{Status: gitenum.FileDiffStatus("weird"), Path: "unknown.txt"},
		},
	}

	info := commitInfoFrom(context.Background(), "space/repo", commit, sv5URLProvider{})
	require.Equal(t, "msg", info.Message)
	require.Equal(t, commit.SHA.String(), info.SHA)
	require.Equal(t, "https://ui/space/repo/commit/"+commit.SHA.String(), info.URL)
	require.Equal(t, []string{"added.txt", "copied.txt", "new.txt"}, info.Added)
	require.Equal(t, []string{"deleted.txt", "old.txt"}, info.Removed)
	require.Equal(t, []string{"modified.txt"}, info.Modified)

	many := commitsInfoFrom(context.Background(), "space/repo", []git.Commit{commit, {}}, sv5URLProvider{})
	require.Len(t, many, 2)
	require.Empty(t, many[1].Added)
}

type sv5ExecutionStore struct {
	store.WebhookExecutionStore

	executions []*types.WebhookExecution
	err        error
}

func (s sv5ExecutionStore) ListForTrigger(context.Context, string) ([]*types.WebhookExecution, error) {
	return s.executions, s.err
}

func TestSv5ExecutorStoreListForTrigger(t *testing.T) {
	ctx := context.Background()

	st := &GitnessWebhookExecutorStore{webhookExecutionStore: sv5ExecutionStore{
		executions: []*types.WebhookExecution{{ID: 1, Error: "e"}, {ID: 2}},
	}}
	got, err := st.ListForTrigger(ctx, "trigger")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, int64(1), got[0].ID)

	st = &GitnessWebhookExecutorStore{webhookExecutionStore: sv5ExecutionStore{err: errSv5Boom}}
	_, err = st.ListForTrigger(ctx, "trigger")
	require.ErrorIs(t, err, errSv5Boom)
}
