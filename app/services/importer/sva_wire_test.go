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

package importer

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func svaNewConfig() *types.Config {
	config := &types.Config{}
	config.Git.DefaultBranch = "trunk"
	return config
}

func TestSvaProvideImporter(t *testing.T) {
	t.Parallel()

	config := svaNewConfig()
	config.Importer.AllowLoopback = true
	config.Importer.AllowLinkLocal = true

	imp := ProvideImporter(
		config, svaURLProvider{}, &svaGit{}, svaTx{}, nil, nil, nil, refcache.RepoFinder{},
		nil, nil, nil, nil, nil, nil,
	)

	require.Equal(t, "trunk", imp.defaultBranch)
	require.Equal(t, netpolicy.Policy{AllowLoopback: true, AllowLinkLocal: true}, imp.networkPolicy)
	require.NotNil(t, imp.baseTransport)
	require.False(t, imp.traceHTTP)
}

func TestSvaProvideNoopServices(t *testing.T) {
	t.Parallel()

	require.IsType(t, connectorServiceNoop{}, ProvideConnectorService())
	require.IsType(t, webhookServiceNoop{}, ProvideWebhookService())
}

func TestSvaProvideJobRepositoryImport(t *testing.T) {
	t.Parallel()

	executor := job.NewExecutor(nil, nil)

	jobRepo, err := ProvideJobRepositoryImport(svaEncrypter{}, nil, executor, svaNewImporter(&svaGit{}))
	require.NoError(t, err)
	require.NotNil(t, jobRepo)

	_, err = ProvideJobRepositoryImport(svaEncrypter{}, nil, executor, nil)
	require.Error(t, err, "the job type can only be registered once")
}

func TestSvaProvideJobReferenceSync(t *testing.T) {
	t.Parallel()

	executor := job.NewExecutor(nil, nil)

	sync, err := ProvideJobReferenceSync(
		svaNewConfig(), svaURLProvider{}, &svaGit{}, nil, refcache.RepoFinder{}, nil, executor, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "trunk", sync.defaultBranch)

	_, err = ProvideJobReferenceSync(
		svaNewConfig(), svaURLProvider{}, &svaGit{}, nil, refcache.RepoFinder{}, nil, executor, nil, nil)
	require.Error(t, err, "the job type can only be registered once")
}

func TestSvaProvideJobRepositoryLink(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("registers the link and the recurring sync job", func(t *testing.T) {
		t.Parallel()

		jobStore := &svaJobStore{}
		scheduler := svaNewScheduler(t, jobStore)
		executor := job.NewExecutor(nil, nil)

		link, err := ProvideJobRepositoryLink(
			ctx, svaNewConfig(), scheduler, executor, svaURLProvider{}, &svaGit{}, connectorServiceNoop{},
			nil, nil, refcache.RepoFinder{}, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, link)

		require.Len(t, jobStore.upserted, 1)
		require.Equal(t, "gitness:jobs:sync_linked_repositories", jobStore.upserted[0].UID)
		require.True(t, jobStore.upserted[0].IsRecurring)
		require.Equal(t, "45 */4 * * *", jobStore.upserted[0].RecurringCron)
	})

	t.Run("link job registered twice", func(t *testing.T) {
		t.Parallel()

		scheduler := svaNewScheduler(t, &svaJobStore{})
		executor := job.NewExecutor(nil, nil)
		require.NoError(t, executor.Register(jobTypeRepositoryLink, &JobRepositoryLink{}))

		_, err := ProvideJobRepositoryLink(
			ctx, svaNewConfig(), scheduler, executor, svaURLProvider{}, &svaGit{}, connectorServiceNoop{},
			nil, nil, refcache.RepoFinder{}, nil, nil, nil)
		require.Error(t, err)
	})

	t.Run("recurring job can't be stored", func(t *testing.T) {
		t.Parallel()

		scheduler := svaNewScheduler(t, &svaJobStore{err: errSvaBoom})
		executor := job.NewExecutor(nil, nil)

		_, err := ProvideJobRepositoryLink(
			ctx, svaNewConfig(), scheduler, executor, svaURLProvider{}, &svaGit{}, connectorServiceNoop{},
			nil, nil, refcache.RepoFinder{}, nil, nil, nil)
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("sync job registered twice", func(t *testing.T) {
		t.Parallel()

		scheduler := svaNewScheduler(t, &svaJobStore{})
		executor := job.NewExecutor(nil, nil)
		require.NoError(t, executor.Register("gitness:jobs:sync_linked_repositories", &JobRepositoryLink{}))

		_, err := ProvideJobRepositoryLink(
			ctx, svaNewConfig(), scheduler, executor, svaURLProvider{}, &svaGit{}, connectorServiceNoop{},
			nil, nil, refcache.RepoFinder{}, nil, nil, nil)
		require.Error(t, err)
	})
}
