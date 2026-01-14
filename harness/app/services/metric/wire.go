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

package metric

import (
	"context"
	"fmt"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	repoevents "github.com/harness/gitness/app/events/repo"
	ruleevents "github.com/harness/gitness/app/events/rule"
	userevents "github.com/harness/gitness/app/events/user"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/job"
	registrystore "github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideValues,
	ProvideSubmitter,
	ProvideCollectorJob,
)

func ProvideValues(ctx context.Context, config *types.Config, settingsSrv *settings.Service) (*Values, error) {
	return NewValues(ctx, config, settingsSrv)
}

func ProvideSubmitter(
	appCtx context.Context,
	config *types.Config,
	values *Values,
	principalStore store.PrincipalStore,
	principalInfoCache store.PrincipalInfoCache,
	pullReqStore store.PullReqStore,
	ruleStore store.RuleStore,
	userEvReaderFactory *events.ReaderFactory[*userevents.Reader],
	repoEvReaderFactory *events.ReaderFactory[*repoevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	ruleEvReaderFactory *events.ReaderFactory[*ruleevents.Reader],
	publicAccess publicaccess.Service,
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
) (Submitter, error) {
	submitter, err := NewPostHog(appCtx, config, values, principalStore, principalInfoCache)
	if err != nil {
		return nil, fmt.Errorf("failed to create posthog metrics submitter: %w", err)
	}

	err = registerEventListeners(
		appCtx,
		config,
		principalInfoCache,
		pullReqStore,
		ruleStore,
		userEvReaderFactory,
		repoEvReaderFactory,
		pullreqEvReaderFactory,
		ruleEvReaderFactory,
		spaceFinder,
		repoFinder,
		publicAccess,
		submitter,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register metric event listeners: %w", err)
	}

	return submitter, nil
}

func ProvideCollectorJob(
	config *types.Config,
	values *Values,
	userStore store.PrincipalStore,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	executionStore store.ExecutionStore,
	scheduler *job.Scheduler,
	executor *job.Executor,
	gitspaceConfigStore store.GitspaceConfigStore,
	registryStore registrystore.RegistryRepository,
	artifactStore registrystore.ArtifactRepository,
	submitter Submitter,
) (*CollectorJob, error) {
	collector := NewCollectorJob(
		values,
		config.Metric.Endpoint,
		config.Metric.Token,
		userStore,
		repoStore,
		pipelineStore,
		executionStore,
		scheduler,
		gitspaceConfigStore,
		registryStore,
		artifactStore,
		submitter,
	)

	err := executor.Register(jobType, collector)
	if err != nil {
		return nil, err
	}

	return collector, nil
}
