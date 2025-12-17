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
	"fmt"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func CreateAndRegisterJobSyncLinkedRepositories(
	ctx context.Context,
	scheduler *job.Scheduler,
	executor *job.Executor,
	defaultBranch string,
	urlProvider gitnessurl.Provider,
	git git.Interface,
	repoFinder refcache.RepoFinder,
	linkedRepoStore store.LinkedRepoStore,
	indexer keywordsearch.Indexer,
	connectorService ConnectorService,
) error {
	const (
		jobMaxDuration = 3*time.Hour + 55*time.Minute
		jobType        = "gitness:jobs:sync_linked_repositories"
		jobUID         = jobType
		jobCron        = "45 */4 * * *" // every 4 hours at 45 minutes
	)

	err := scheduler.AddRecurring(
		ctx,
		jobUID,
		jobType,
		jobCron,
		jobMaxDuration)
	if err != nil {
		return fmt.Errorf("failed to create recurring job linked repositories sync: %w", err)
	}

	handler := NewJobSyncLinkedRepositories(
		defaultBranch,
		urlProvider,
		git,
		repoFinder,
		linkedRepoStore,
		scheduler,
		indexer,
		connectorService,
	)
	err = executor.Register(jobType, handler)
	if err != nil {
		return err
	}

	return nil
}

func NewJobSyncLinkedRepositories(
	defaultBranch string,
	urlProvider gitnessurl.Provider,
	git git.Interface,
	repoFinder refcache.RepoFinder,
	linkedRepoStore store.LinkedRepoStore,
	scheduler *job.Scheduler,
	indexer keywordsearch.Indexer,
	connectorService ConnectorService,
) *JobSyncLinkedRepositories {
	return &JobSyncLinkedRepositories{
		defaultBranch:    defaultBranch,
		urlProvider:      urlProvider,
		git:              git,
		repoFinder:       repoFinder,
		linkedRepoStore:  linkedRepoStore,
		scheduler:        scheduler,
		indexer:          indexer,
		connectorService: connectorService,
	}
}

type JobSyncLinkedRepositories struct {
	defaultBranch    string
	urlProvider      gitnessurl.Provider
	git              git.Interface
	repoFinder       refcache.RepoFinder
	linkedRepoStore  store.LinkedRepoStore
	scheduler        *job.Scheduler
	indexer          keywordsearch.Indexer
	connectorService ConnectorService
}

var _ job.Handler = (*JobSyncLinkedRepositories)(nil)

type JobLinkedRepositoriesSyncInput struct {
	SourceRepoID int64       `json:"source_repo_id"`
	TargetRepoID int64       `json:"target_repo_id"`
	RefSpecType  RefSpecType `json:"ref_spec_type"`
	SourceRef    string      `json:"source_ref"`
	TargetRef    string      `json:"target_ref"`
}

// Handle executes synchronization of linked repositories.
func (r *JobSyncLinkedRepositories) Handle(
	ctx context.Context,
	_ string,
	progress job.ProgressReporter,
) (string, error) {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal

	refSpec := []string{
		api.BranchPrefix + "*:" + api.BranchPrefix + "*",
		api.TagPrefix + "*:" + api.TagPrefix + "*",
	}

	const limit = 1000

	linkedRepos, err := r.linkedRepoStore.List(ctx, limit)
	if err != nil {
		return "", fmt.Errorf("failed to list linked repositories: %w", err)
	}

	for linkedRepoIndex, linkedRepo := range linkedRepos {
		log := log.Ctx(ctx).With().
			Int64("repo.id", linkedRepo.RepoID).
			Logger()

		repo, err := r.repoFinder.FindByID(ctx, linkedRepo.RepoID)
		if err != nil {
			log.Warn().Err(err).Msg("failed to find repo")
			continue
		}

		connector := ConnectorDef{
			Path:       linkedRepo.ConnectorPath,
			Identifier: linkedRepo.ConnectorIdentifier,
		}

		accessInfo, err := r.connectorService.GetAccessInfo(ctx, connector)
		if err != nil {
			log.Warn().Err(err).Msg("failed to access info from connector")
			continue
		}

		cloneURLWithAuth, err := accessInfo.URLWithCredentials()
		if err != nil {
			log.Warn().Err(err).Msg("failed to get clone URL from connector's access info")
			continue
		}

		writeParams, err := r.createRPCWriteParams(ctx, systemPrincipal, repo.ID, repo.GitUID)
		if err != nil {
			return "", fmt.Errorf("failed to create rpc write params: %w", err)
		}

		_, err = r.git.SyncRepository(ctx, &git.SyncRepositoryParams{
			WriteParams:       writeParams,
			Source:            cloneURLWithAuth,
			CreateIfNotExists: false,
			RefSpecs:          refSpec,
		})
		if err != nil {
			return "", fmt.Errorf("failed to sync repository: %w", err)
		}

		_, err = r.linkedRepoStore.UpdateOptLock(ctx, &linkedRepo, func(l *types.LinkedRepo) error {
			l.LastFullSync = time.Now().UnixMilli()
			return nil
		})
		if err != nil {
			log.Warn().Err(err).Msg("failed to update linked repo")
			continue
		}

		log.Info().Msg("synced linked repository")

		err = progress(100*linkedRepoIndex/len(linkedRepos), "")
		if err != nil {
			log.Warn().Err(err).Msg("failed to update job progress")
			continue
		}
	}

	return "", nil
}

func (r *JobSyncLinkedRepositories) createRPCWriteParams(
	ctx context.Context,
	principal types.Principal,
	repoID int64,
	repoGitUID string,
) (git.WriteParams, error) {
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		r.urlProvider.GetInternalAPIURL(ctx),
		repoID,
		principal.ID,
		true,
		true,
	)
	if err != nil {
		return git.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return git.WriteParams{
		RepoUID: repoGitUID,
		Actor: git.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		EnvVars: envVars,
	}, nil
}
