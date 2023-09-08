// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package importer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/internal/store"
	gitnessurl "github.com/harness/gitness/internal/url"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

type Repository struct {
	urlProvider *gitnessurl.Provider
	git         gitrpc.Interface
	repoStore   store.RepoStore
	scheduler   *job.Scheduler
}

var _ job.Handler = (*Repository)(nil)

type Input struct {
	RepoID   int64  `json:"repo_id"`
	GitUser  string `json:"git_user"`
	GitPass  string `json:"git_pass"`
	CloneURL string `json:"clone_url"`
}

const jobType = "repository_import"

func (i *Repository) Register(executor *job.Executor) error {
	return executor.Register(jobType, i)
}

func (i *Repository) Run(ctx context.Context, jobUID string, input Input) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}

	strData := strings.TrimSpace(string(data))

	return i.scheduler.RunJob(ctx, job.Definition{
		UID:        jobUID,
		Type:       jobType,
		MaxRetries: 1,
		Timeout:    30 * time.Minute,
		Data:       strData,
	})
}

// Handle is repository import background job handler.
func (i *Repository) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	var input Input
	if err := json.NewDecoder(strings.NewReader(data)).Decode(&input); err != nil {
		return "", fmt.Errorf("failed to unmarshal job input: %w", err)
	}

	if input.CloneURL == "" {
		return "", errors.New("missing git repository clone URL")
	}

	if input.GitUser != "" || input.GitPass != "" {
		repoURL, err := url.Parse(input.CloneURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse git clone URL: %w", err)
		}

		repoURL.User = url.UserPassword(input.GitUser, input.GitPass)
		input.CloneURL = repoURL.String()
	}

	repo, err := i.repoStore.Find(ctx, input.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to find repo by id: %w", err)
	}

	if !repo.Importing {
		return "", fmt.Errorf("repository %s is not being imported", repo.UID)
	}

	writeParams, err := createRPCWriteParams(ctx, i.urlProvider, repo)
	if err != nil {
		return "", fmt.Errorf("failed to create write params: %w", err)
	}

	syncOut, err := i.git.SyncRepository(ctx, &gitrpc.SyncRepositoryParams{
		WriteParams:       writeParams,
		Source:            input.CloneURL,
		CreateIfNotExists: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to sync repositories: %w", err)
	}

	repo.Importing = false
	repo.DefaultBranch = syncOut.DefaultBranch

	err = i.repoStore.Update(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("failed to update repository after import: %w", err)
	}

	return "", err
}

func (i *Repository) GetProgress(ctx context.Context, repo *types.Repository) (types.JobProgress, error) {
	if !repo.Importing || repo.ImportingJobUID == nil || *repo.ImportingJobUID == "" {
		// if the repo is not being imported, or it's job ID has been cleared (or never existed) return state=finished
		return job.DoneProgress(), nil
	}

	progress, err := i.scheduler.GetJobProgress(ctx, *repo.ImportingJobUID)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		// if the job is not found return state=failed
		return job.FailProgress(), nil
	}
	if err != nil {
		return types.JobProgress{}, fmt.Errorf("failed to get job progress: %w", err)
	}

	return progress, nil
}

// CreateRPCWriteParams creates base write parameters for gitrpc write operations.
func createRPCWriteParams(ctx context.Context,
	urlProvider *gitnessurl.Provider,
	repo *types.Repository,
) (gitrpc.WriteParams, error) {
	gitnessSession := bootstrap.NewSystemServiceSession()

	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		urlProvider.GetAPIBaseURLInternal(),
		repo.ID,
		gitnessSession.Principal.ID,
		false,
	)
	if err != nil {
		return gitrpc.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return gitrpc.WriteParams{
		Actor: gitrpc.Identity{
			Name:  gitnessSession.Principal.DisplayName,
			Email: gitnessSession.Principal.Email,
		},
		RepoUID: repo.GitUID,
		EnvVars: envVars,
	}, nil
}
