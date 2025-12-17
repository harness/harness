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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	jobLinkRepoMaxRetries  = 0
	jobLinkRepoMaxDuration = 45 * time.Minute
)

type JobRepositoryLink struct {
	scheduler        *job.Scheduler
	urlProvider      gitnessurl.Provider
	git              git.Interface
	connectorService ConnectorService
	repoStore        store.RepoStore
	linkedRepoStore  store.LinkedRepoStore
	repoFinder       refcache.RepoFinder
	sseStreamer      sse.Streamer
	indexer          keywordsearch.Indexer
	eventReporter    *repoevents.Reporter
}

var _ job.Handler = (*JobRepositoryLink)(nil)

func NewJobRepositoryLink(
	scheduler *job.Scheduler,
	urlProvider gitnessurl.Provider,
	git git.Interface,
	connectorService ConnectorService,
	repoStore store.RepoStore,
	linkedRepoStore store.LinkedRepoStore,
	repoFinder refcache.RepoFinder,
	sseStreamer sse.Streamer,
	indexer keywordsearch.Indexer,
	eventReporter *repoevents.Reporter,
) *JobRepositoryLink {
	return &JobRepositoryLink{
		scheduler:        scheduler,
		urlProvider:      urlProvider,
		git:              git,
		connectorService: connectorService,
		repoStore:        repoStore,
		linkedRepoStore:  linkedRepoStore,
		repoFinder:       repoFinder,
		sseStreamer:      sseStreamer,
		indexer:          indexer,
		eventReporter:    eventReporter,
	}
}

type JobLinkRepoInput struct {
	RepoID   int64 `json:"repo_id"`
	IsPublic bool  `json:"is_public"`
}

const jobTypeRepositoryLink = "link_repository_import"

func (r *JobRepositoryLink) Register(executor *job.Executor) error {
	return executor.Register(jobTypeRepositoryLink, r)
}

// Run starts a background job that imports the provided repository from the provided clone URL.
func (r *JobRepositoryLink) Run(ctx context.Context, repoID int64, isPublic bool) error {
	jobID := r.jobIDFromRepoID(repoID)
	jobDef, err := r.getJobDef(jobID, JobLinkRepoInput{
		RepoID:   repoID,
		IsPublic: isPublic,
	})
	if err != nil {
		return err
	}

	return r.scheduler.RunJob(ctx, jobDef)
}

func (*JobRepositoryLink) jobIDFromRepoID(repoID int64) string {
	const jobIDPrefix = "link-repo-"
	return jobIDPrefix + strconv.FormatInt(repoID, 10)
}

func (r *JobRepositoryLink) getJobDef(jobUID string, input JobLinkRepoInput) (job.Definition, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return job.Definition{}, fmt.Errorf("failed to marshal job input json: %w", err)
	}

	return job.Definition{
		UID:        jobUID,
		Type:       jobTypeRepositoryLink,
		MaxRetries: jobLinkRepoMaxRetries,
		Timeout:    jobLinkRepoMaxDuration,
		Data:       base64.StdEncoding.EncodeToString(bytes.TrimSpace(data)),
	}, nil
}

func (r *JobRepositoryLink) getJobInput(data string) (JobLinkRepoInput, error) {
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return JobLinkRepoInput{}, fmt.Errorf("failed to base64 decode job input: %w", err)
	}

	var input JobLinkRepoInput

	err = json.NewDecoder(bytes.NewReader(raw)).Decode(&input)
	if err != nil {
		return JobLinkRepoInput{}, fmt.Errorf("failed to unmarshal job input json: %w", err)
	}

	return input, nil
}

func (r *JobRepositoryLink) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	input, err := r.getJobInput(data)
	if err != nil {
		return "", err
	}

	systemPrincipal := bootstrap.NewSystemServiceSession().Principal
	gitIdentity := git.Identity{
		Name:  systemPrincipal.DisplayName,
		Email: systemPrincipal.Email,
	}

	repo, err := r.repoStore.Find(ctx, input.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to find repo by id: %w", err)
	}

	linkedRepo, err := r.linkedRepoStore.Find(ctx, repo.ID)
	if err != nil {
		return "", fmt.Errorf("failed to find linked repo by repo id: %w", err)
	}

	accessInfo, err := r.connectorService.GetAccessInfo(ctx, ConnectorDef{
		Path:       linkedRepo.ConnectorPath,
		Identifier: linkedRepo.ConnectorIdentifier,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get repository access info from connector: %w", err)
	}

	cloneURLWithAuth, err := accessInfo.URLWithCredentials()
	if err != nil {
		return "", fmt.Errorf("failed to parse git clone URL: %w", err)
	}

	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		r.urlProvider.GetInternalAPIURL(ctx),
		repo.ID,
		systemPrincipal.ID,
		true,
		true,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create environment variables: %w", err)
	}

	if repo.State != enum.RepoStateGitImport {
		return "", fmt.Errorf("repository %s is not being imported", repo.Identifier)
	}

	log := log.Ctx(ctx).With().
		Int64("repo.id", repo.ID).
		Str("repo.path", repo.Path).
		Logger()

	now := time.Now()

	respCreateRepo, err := r.git.CreateRepository(ctx, &git.CreateRepositoryParams{
		Actor:         gitIdentity,
		EnvVars:       envVars,
		DefaultBranch: repo.DefaultBranch,
		Files:         nil,
		Author:        &gitIdentity,
		AuthorDate:    &now,
		Committer:     &gitIdentity,
		CommitterDate: &now,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create empty git repository: %w", err)
	}

	gitUID := respCreateRepo.UID

	writeParams := git.WriteParams{
		Actor:   gitIdentity,
		RepoUID: gitUID,
		EnvVars: envVars,
	}

	err = func() error {
		_, err = r.git.SyncRepository(ctx, &git.SyncRepositoryParams{
			WriteParams:       writeParams,
			Source:            cloneURLWithAuth,
			CreateIfNotExists: false,
			RefSpecs:          []string{"refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"},
			DefaultBranch:     repo.DefaultBranch,
		})
		if err != nil {
			return fmt.Errorf("failed to sync repository: %w", err)
		}

		repo, err = r.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
			if repo.State != enum.RepoStateGitImport {
				return errors.New("repository has already finished importing")
			}

			repo.State = enum.RepoStateActive
			repo.GitUID = gitUID

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update repository after import: %w", err)
		}

		r.repoFinder.MarkChanged(ctx, repo.Core())

		return nil
	}()
	if err != nil {
		log.Error().Err(err).Msg("failed repository import - cleanup git repository")

		if errDel := r.git.DeleteRepository(context.WithoutCancel(ctx), &git.DeleteRepositoryParams{
			WriteParams: writeParams,
		}); errDel != nil {
			log.Warn().Err(errDel).
				Msg("failed to delete git repository after failed import")
		}

		return "", fmt.Errorf("failed to import repository: %w", err)
	}

	r.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypeRepositoryImportCompleted, repo)

	r.eventReporter.Created(ctx, &repoevents.CreatedPayload{
		Base: repoevents.Base{
			RepoID:      repo.ID,
			PrincipalID: bootstrap.NewSystemServiceSession().Principal.ID,
		},
		IsPublic:     input.IsPublic,
		ImportedFrom: repo.GitURL,
	})

	err = r.indexer.Index(ctx, repo)
	if err != nil {
		log.Warn().Err(err).Msg("failed to index repository")
	}

	return "", nil
}
