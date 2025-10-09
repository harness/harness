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
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	refSyncJobMaxRetries  = 2
	refSyncJobMaxDuration = 45 * time.Minute
	refSyncJobType        = "reference_sync"
)

type ReferenceSync struct {
	defaultBranch string
	urlProvider   gitnessurl.Provider
	git           git.Interface
	repoStore     store.RepoStore
	repoFinder    refcache.RepoFinder
	scheduler     *job.Scheduler
	indexer       keywordsearch.Indexer
	eventReporter *repoevents.Reporter
}

var _ job.Handler = (*ReferenceSync)(nil)

type RefSpecType string

const (
	RefSpecTypeReference       RefSpecType = "reference"
	RefSpecTypeDefaultBranch   RefSpecType = "default_branch"
	RefSpecTypeAllBranches     RefSpecType = "all_branches"
	RefSpecTypeBranchesAndTags RefSpecType = "branches_and_tags"
)

type ReferenceSyncInput struct {
	SourceRepoID int64       `json:"source_repo_id"`
	TargetRepoID int64       `json:"target_repo_id"`
	RefSpecType  RefSpecType `json:"ref_spec_type"`
	SourceRef    string      `json:"source_ref"`
	TargetRef    string      `json:"target_ref"`
}

func (r *ReferenceSync) Register(executor *job.Executor) error {
	return executor.Register(refSyncJobType, r)
}

// Run starts a background job that imports the provided repository from the provided clone URL.
func (r *ReferenceSync) Run(
	ctx context.Context,
	sourceRepoID, targetRepoID int64,
	refSpecType RefSpecType,
	sourceRef, targetRef string,
) error {
	var idRaw [10]byte
	if _, err := rand.Read(idRaw[:]); err != nil {
		return fmt.Errorf("could not generate repository sync job ID: %w", err)
	}

	id := base32.StdEncoding.EncodeToString(idRaw[:])

	jobDef, err := r.getJobDef(id, ReferenceSyncInput{
		SourceRepoID: sourceRepoID,
		TargetRepoID: targetRepoID,
		RefSpecType:  refSpecType,
		SourceRef:    sourceRef,
		TargetRef:    targetRef,
	})
	if err != nil {
		return fmt.Errorf("could not get repository sync job definition: %w", err)
	}

	return r.scheduler.RunJob(ctx, jobDef)
}

func (r *ReferenceSync) getJobDef(jobUID string, input ReferenceSyncInput) (job.Definition, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return job.Definition{}, fmt.Errorf("failed to marshal repository sync job input json: %w", err)
	}

	data = bytes.TrimSpace(data)

	return job.Definition{
		UID:        jobUID,
		Type:       refSyncJobType,
		MaxRetries: refSyncJobMaxRetries,
		Timeout:    refSyncJobMaxDuration,
		Data:       string(data),
	}, nil
}

func (r *ReferenceSync) getJobInput(data string) (ReferenceSyncInput, error) {
	var input ReferenceSyncInput

	err := json.NewDecoder(strings.NewReader(data)).Decode(&input)
	if err != nil {
		return ReferenceSyncInput{}, fmt.Errorf("failed to unmarshal repository sync job input json: %w", err)
	}

	return input, nil
}

// Handle is repository import background job handler.
//
//nolint:gocognit // refactor if needed.
func (r *ReferenceSync) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal

	input, err := r.getJobInput(data)
	if err != nil {
		return "", err
	}

	repoSource, err := r.repoFinder.FindByID(ctx, input.SourceRepoID)
	if err != nil {
		return "", fmt.Errorf("failed to find source repo by id: %w", err)
	}

	repoTarget, err := r.repoFinder.FindByID(ctx, input.TargetRepoID)
	if err != nil {
		return "", fmt.Errorf("failed to find target repo by id: %w", err)
	}

	writeParams, err := r.createRPCWriteParams(ctx, systemPrincipal, repoTarget.ID, repoTarget.GitUID)
	if err != nil {
		return "", fmt.Errorf("failed to create rpc write params: %w", err)
	}

	log := log.Ctx(ctx).With().
		Int64("repo_source.id", repoSource.ID).
		Str("repo_source.path", repoSource.Path).
		Int64("repo_target.id", repoTarget.ID).
		Str("repo_target.path", repoTarget.Path).
		Logger()

	var refSpec []string
	var needIndexing bool

	switch input.RefSpecType {
	case RefSpecTypeReference:
		refSpec = []string{input.SourceRef + ":" + input.TargetRef}
		needIndexing = input.TargetRef == api.BranchPrefix+repoTarget.DefaultBranch
	case RefSpecTypeDefaultBranch:
		refSpec = []string{
			api.BranchPrefix + repoSource.DefaultBranch + ":" + api.BranchPrefix + repoTarget.DefaultBranch,
		}
		needIndexing = true
	case RefSpecTypeAllBranches:
		refSpec = []string{api.BranchPrefix + "*:" + api.BranchPrefix + "*"}
		needIndexing = true
	case RefSpecTypeBranchesAndTags:
		refSpec = []string{
			api.BranchPrefix + "*:" + api.BranchPrefix + "*",
			api.TagPrefix + "*:" + api.TagPrefix + "*",
		}
		needIndexing = true
	}

	_, err = r.git.SyncRepository(ctx, &git.SyncRepositoryParams{
		WriteParams:       writeParams,
		Source:            repoSource.GitUID,
		CreateIfNotExists: false,
		RefSpecs:          refSpec,
		DefaultBranch:     repoTarget.DefaultBranch,
	})
	if err != nil {
		return "", fmt.Errorf("failed to sync repository: %w", err)
	}

	repoTargetFull, err := r.repoStore.Find(ctx, repoTarget.ID)
	if err != nil {
		return "", fmt.Errorf("failed to find repository: %w", err)
	}

	// Clear the git import status if set

	errNoChange := errors.New("no change")
	repoTargetFull, err = r.repoStore.UpdateOptLock(ctx, repoTargetFull, func(r *types.Repository) error {
		if r.State != enum.RepoStateGitImport {
			return errNoChange
		}
		r.State = enum.RepoStateActive
		return nil
	})
	if err != nil && !errors.Is(err, errNoChange) {
		return "", fmt.Errorf("failed to update repo state: %w", err)
	}

	r.repoFinder.MarkChanged(ctx, repoTargetFull.Core())

	if needIndexing {
		err = r.indexer.Index(ctx, repoTargetFull)
		if err != nil {
			log.Warn().Err(err).Msg("failed to index repository")
		}
	}

	log.Info().Msg("completed repository reference sync job")

	return "", nil
}

func (r *ReferenceSync) createRPCWriteParams(
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
