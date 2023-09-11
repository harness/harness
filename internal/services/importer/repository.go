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

	"github.com/rs/zerolog/log"
)

const (
	importJobMaxRetries  = 0
	importJobMaxDuration = 45 * time.Minute
)

type Repository struct {
	defaultBranch string
	urlProvider   *gitnessurl.Provider
	git           gitrpc.Interface
	repoStore     store.RepoStore
	scheduler     *job.Scheduler
}

var _ job.Handler = (*Repository)(nil)

type Input struct {
	RepoID   int64  `json:"repo_id"`
	GitUser  string `json:"git_user"`
	GitPass  string `json:"git_pass"`
	CloneURL string `json:"clone_url"`
}

const jobType = "repository_import"

func (r *Repository) Register(executor *job.Executor) error {
	return executor.Register(jobType, r)
}

// Run starts a background job that imports the provided repository from the provided clone URL.
func (r *Repository) Run(ctx context.Context, provider Provider, repo *types.Repository, cloneURL string) error {
	input := Input{
		RepoID:   repo.ID,
		GitUser:  provider.Username,
		GitPass:  provider.Password,
		CloneURL: cloneURL,
	}

	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal job input json: %w", err)
	}

	strData := strings.TrimSpace(string(data))

	return r.scheduler.RunJob(ctx, job.Definition{
		UID:        *repo.ImportingJobUID,
		Type:       jobType,
		MaxRetries: importJobMaxRetries,
		Timeout:    importJobMaxDuration,
		Data:       strData,
	})
}

// RunMany starts background jobs that import the provided repositories from the provided clone URLs.
func (r *Repository) RunMany(ctx context.Context,
	groupID string,
	provider Provider,
	repos []*types.Repository,
	cloneURLs []string,
) error {
	if len(repos) != len(cloneURLs) {
		return fmt.Errorf("slice length mismatch: have %d repositories and %d clone URLs",
			len(repos), len(cloneURLs))
	}

	n := len(repos)

	defs := make([]job.Definition, n)

	for k := 0; k < n; k++ {
		repo := repos[k]
		cloneURL := cloneURLs[k]

		input := Input{
			RepoID:   repo.ID,
			GitUser:  provider.Username,
			GitPass:  provider.Password,
			CloneURL: cloneURL,
		}

		data, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("failed to marshal job input json: %w", err)
		}

		strData := strings.TrimSpace(string(data))

		defs[k] = job.Definition{
			UID:        *repo.ImportingJobUID,
			Type:       jobType,
			MaxRetries: importJobMaxRetries,
			Timeout:    importJobMaxDuration,
			Data:       strData,
		}
	}

	err := r.scheduler.RunJobs(ctx, groupID, defs)
	if err != nil {
		return fmt.Errorf("failed to run jobs: %w", err)
	}

	return nil
}

// Handle is repository import background job handler.
func (r *Repository) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal

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

	repo, err := r.repoStore.Find(ctx, input.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to find repo by id: %w", err)
	}

	if !repo.Importing {
		return "", fmt.Errorf("repository %s is not being imported", repo.UID)
	}

	gitUID, err := r.createGitRepository(ctx, &systemPrincipal, repo.ID)
	if err != nil {
		return "", fmt.Errorf("failed to create empty git repository: %w", err)
	}

	err = func() error {
		repo.GitUID = gitUID

		defaultBranch, err := r.syncGitRepository(ctx, &systemPrincipal, repo, input.CloneURL)
		if err != nil {
			return fmt.Errorf("failed to sync git repository from '%s': %w", input.CloneURL, err)
		}

		if defaultBranch == "" {
			defaultBranch = r.defaultBranch
		}

		repo, err = r.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
			if !repo.Importing {
				return errors.New("repository has already finished importing")
			}

			repo.GitUID = gitUID
			repo.DefaultBranch = defaultBranch
			repo.Importing = false

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update repository after import: %w", err)
		}

		return nil
	}()
	if err != nil {
		if errDel := r.deleteGitRepository(ctx, &systemPrincipal, repo); errDel != nil {
			log.Ctx(ctx).Err(err).
				Str("gitUID", gitUID).
				Msg("failed to delete git repository after failed import")
		}

		return "", fmt.Errorf("failed to import repository: %w", err)
	}

	return "", nil
}

func (r *Repository) GetProgress(ctx context.Context, repo *types.Repository) (types.JobProgress, error) {
	if !repo.Importing || repo.ImportingJobUID == nil || *repo.ImportingJobUID == "" {
		// if the repo is not being imported, or it's job ID has been cleared (or never existed) return state=finished
		return job.DoneProgress(), nil
	}

	progress, err := r.scheduler.GetJobProgress(ctx, *repo.ImportingJobUID)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		// if the job is not found return state=failed
		return job.FailProgress(), nil
	}
	if err != nil {
		return types.JobProgress{}, fmt.Errorf("failed to get job progress: %w", err)
	}

	return progress, nil
}

func (r *Repository) Cancel(ctx context.Context, repo *types.Repository) error {
	if repo.ImportingJobUID == nil || *repo.ImportingJobUID == "" {
		return nil
	}

	err := r.scheduler.CancelJob(ctx, *repo.ImportingJobUID)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	return nil
}

func (r *Repository) createGitRepository(ctx context.Context,
	principal *types.Principal,
	repoID int64,
) (string, error) {
	now := time.Now()

	envVars, err := r.createEnvVars(ctx, principal, repoID)
	if err != nil {
		return "", err
	}

	resp, err := r.git.CreateRepository(ctx, &gitrpc.CreateRepositoryParams{
		Actor: gitrpc.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		EnvVars:       envVars,
		DefaultBranch: r.defaultBranch,
		Files:         nil,
		Author: &gitrpc.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		AuthorDate: &now,
		Committer: &gitrpc.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		CommitterDate: &now,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create empty git repository: %w", err)
	}

	return resp.UID, nil
}

func (r *Repository) syncGitRepository(ctx context.Context,
	principal *types.Principal,
	repo *types.Repository,
	sourceCloneURL string,
) (string, error) {
	writeParams, err := r.createRPCWriteParams(ctx, principal, repo)
	if err != nil {
		return "", err
	}

	syncOut, err := r.git.SyncRepository(ctx, &gitrpc.SyncRepositoryParams{
		WriteParams:       writeParams,
		Source:            sourceCloneURL,
		CreateIfNotExists: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to sync repository: %w", err)
	}

	return syncOut.DefaultBranch, nil
}

func (r *Repository) deleteGitRepository(ctx context.Context,
	principal *types.Principal,
	repo *types.Repository,
) error {
	writeParams, err := r.createRPCWriteParams(ctx, principal, repo)
	if err != nil {
		return err
	}

	err = r.git.DeleteRepository(ctx, &gitrpc.DeleteRepositoryParams{
		WriteParams: writeParams,
	})
	if err != nil {
		return fmt.Errorf("failed to delete git repository: %w", err)
	}

	return nil
}

func (r *Repository) createRPCWriteParams(ctx context.Context,
	principal *types.Principal,
	repo *types.Repository,
) (gitrpc.WriteParams, error) {
	envVars, err := r.createEnvVars(ctx, principal, repo.ID)
	if err != nil {
		return gitrpc.WriteParams{}, err
	}

	return gitrpc.WriteParams{
		Actor: gitrpc.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		RepoUID: repo.GitUID,
		EnvVars: envVars,
	}, nil
}

func (r *Repository) createEnvVars(ctx context.Context,
	principal *types.Principal,
	repoID int64,
) (map[string]string, error) {
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		r.urlProvider.GetAPIBaseURLInternal(),
		repoID,
		principal.ID,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return envVars, nil
}
