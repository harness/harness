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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	importJobMaxRetries  = 0
	importJobMaxDuration = 45 * time.Minute
)

var (
	// ErrNotFound is returned if no import data was found.
	ErrNotFound = errors.New("import not found")
)

type Repository struct {
	defaultBranch string
	urlProvider   gitnessurl.Provider
	git           git.Interface
	tx            dbtx.Transactor
	repoStore     store.RepoStore
	pipelineStore store.PipelineStore
	triggerStore  store.TriggerStore
	encrypter     encrypt.Encrypter
	scheduler     *job.Scheduler
	sseStreamer   sse.Streamer
	indexer       keywordsearch.Indexer
	publicAccess  publicaccess.Service
	auditService  audit.Service
}

var _ job.Handler = (*Repository)(nil)

// PipelineOption defines the supported pipeline import options for repository import.
type PipelineOption string

func (PipelineOption) Enum() []any {
	return []any{PipelineOptionConvert, PipelineOptionIgnore}
}

const (
	PipelineOptionConvert PipelineOption = "convert"
	PipelineOptionIgnore  PipelineOption = "ignore"
)

type Input struct {
	RepoID    int64          `json:"repo_id"`
	Public    bool           `json:"public"`
	GitUser   string         `json:"git_user"`
	GitPass   string         `json:"git_pass"`
	CloneURL  string         `json:"clone_url"`
	Pipelines PipelineOption `json:"pipelines"`
}

const jobType = "repository_import"

func (r *Repository) Register(executor *job.Executor) error {
	return executor.Register(jobType, r)
}

// Run starts a background job that imports the provided repository from the provided clone URL.
func (r *Repository) Run(
	ctx context.Context,
	provider Provider,
	repo *types.Repository,
	public bool,
	cloneURL string,
	pipelines PipelineOption,
) error {
	jobDef, err := r.getJobDef(JobIDFromRepoID(repo.ID), Input{
		RepoID:    repo.ID,
		Public:    public,
		GitUser:   provider.Username,
		GitPass:   provider.Password,
		CloneURL:  cloneURL,
		Pipelines: pipelines,
	})
	if err != nil {
		return err
	}

	return r.scheduler.RunJob(ctx, jobDef)
}

// RunMany starts background jobs that import the provided repositories from the provided clone URLs.
func (r *Repository) RunMany(ctx context.Context,
	groupID string,
	provider Provider,
	repoIDs []int64,
	publics []bool,
	cloneURLs []string,
	pipelines PipelineOption,
) error {
	if len(repoIDs) != len(cloneURLs) {
		return fmt.Errorf("slice length mismatch: have %d repositories and %d clone URLs",
			len(repoIDs), len(cloneURLs))
	}

	n := len(repoIDs)
	defs := make([]job.Definition, n)

	for k := 0; k < n; k++ {
		repoID := repoIDs[k]
		cloneURL := cloneURLs[k]

		jobDef, err := r.getJobDef(JobIDFromRepoID(repoID), Input{
			RepoID:    repoID,
			Public:    publics[k],
			GitUser:   provider.Username,
			GitPass:   provider.Password,
			CloneURL:  cloneURL,
			Pipelines: pipelines,
		})
		if err != nil {
			return err
		}

		defs[k] = jobDef
	}

	err := r.scheduler.RunJobs(ctx, groupID, defs)
	if err != nil {
		return fmt.Errorf("failed to run jobs: %w", err)
	}

	return nil
}

func (r *Repository) getJobDef(jobUID string, input Input) (job.Definition, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return job.Definition{}, fmt.Errorf("failed to marshal job input json: %w", err)
	}

	strData := strings.TrimSpace(string(data))

	encryptedData, err := r.encrypter.Encrypt(strData)
	if err != nil {
		return job.Definition{}, fmt.Errorf("failed to encrypt job input: %w", err)
	}

	return job.Definition{
		UID:        jobUID,
		Type:       jobType,
		MaxRetries: importJobMaxRetries,
		Timeout:    importJobMaxDuration,
		Data:       base64.StdEncoding.EncodeToString(encryptedData),
	}, nil
}

func (r *Repository) getJobInput(data string) (Input, error) {
	encrypted, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return Input{}, fmt.Errorf("failed to base64 decode job input: %w", err)
	}

	decrypted, err := r.encrypter.Decrypt(encrypted)
	if err != nil {
		return Input{}, fmt.Errorf("failed to decrypt job input: %w", err)
	}

	var input Input

	err = json.NewDecoder(strings.NewReader(decrypted)).Decode(&input)
	if err != nil {
		return Input{}, fmt.Errorf("failed to unmarshal job input json: %w", err)
	}

	return input, nil
}

// Handle is repository import background job handler.
//
//nolint:gocognit // refactor if needed.
func (r *Repository) Handle(ctx context.Context, data string, _ job.ProgressReporter) (string, error) {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal

	input, err := r.getJobInput(data)
	if err != nil {
		return "", err
	}

	if input.CloneURL == "" {
		return "", errors.New("missing git repository clone URL")
	}

	repoURL, err := url.Parse(input.CloneURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse git clone URL: %w", err)
	}

	repoURL.User = url.UserPassword(input.GitUser, input.GitPass)
	cloneURLWithAuth := repoURL.String()

	repo, err := r.repoStore.Find(ctx, input.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to find repo by id: %w", err)
	}

	if repo.State != enum.RepoStateGitImport {
		return "", fmt.Errorf("repository %s is not being imported", repo.Identifier)
	}

	log := log.Ctx(ctx).With().
		Int64("repo.id", repo.ID).
		Str("repo.path", repo.Path).
		Logger()

	log.Info().Msg("configure access mode")

	parentPath, _, err := paths.DisectLeaf(repo.Path)
	if err != nil {
		return "", fmt.Errorf("failed to disect path %q: %w", repo.Path, err)
	}
	isPublicAccessSupported, err := r.publicAccess.IsPublicAccessSupported(ctx, parentPath)
	if err != nil {
		return "", fmt.Errorf(
			"failed to check if public access is supported for parent space %q: %w",
			parentPath,
			err,
		)
	}
	isRepoPublic := input.Public
	if !isPublicAccessSupported {
		log.Debug().Msg("public access is not supported, import public repo as private instead")
		isRepoPublic = false
	}
	err = r.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, repo.Path, isRepoPublic)
	if err != nil {
		return "", fmt.Errorf("failed to set repo access mode: %w", err)
	}

	if isRepoPublic {
		err = r.auditService.Log(ctx,
			bootstrap.NewPipelineServiceSession().Principal,
			audit.NewResource(audit.ResourceTypeRepository, repo.Identifier),
			audit.ActionUpdated,
			paths.Parent(repo.Path),
			audit.WithOldObject(audit.RepositoryObject{
				Repository: *repo,
				IsPublic:   false,
			}),
			audit.WithNewObject(audit.RepositoryObject{
				Repository: *repo,
				IsPublic:   true,
			}),
		)
		if err != nil {
			log.Warn().Msgf("failed to insert audit log for updating repo to public: %s", err)
		}
	}

	log.Info().Msg("create git repository")

	gitUID, err := r.createGitRepository(ctx, &systemPrincipal, repo.ID)
	if err != nil {
		return "", fmt.Errorf("failed to create empty git repository: %w", err)
	}

	log.Info().Msgf("successfully created git repository with git_uid '%s'", gitUID)

	err = func() error {
		repo, err = r.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
			if repo.State != enum.RepoStateGitImport {
				return errors.New("repository has already finished importing")
			}
			repo.GitUID = gitUID
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update repository prior to the import: %w", err)
		}

		log.Info().Msg("sync repository")

		defaultBranch, err := r.syncGitRepository(ctx, &systemPrincipal, repo, cloneURLWithAuth)
		if err != nil {
			return fmt.Errorf("failed to sync git repository from '%s': %w", input.CloneURL, err)
		}

		log.Info().Msgf("successfully synced repository (returned default branch: '%s')", defaultBranch)

		if defaultBranch == "" {
			defaultBranch = r.defaultBranch
		}

		log.Info().Msg("update repo in DB")

		repo, err = r.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
			if repo.State != enum.RepoStateGitImport {
				return errors.New("repository has already finished importing")
			}

			repo.GitUID = gitUID
			repo.DefaultBranch = defaultBranch
			repo.State = enum.RepoStateActive

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update repository after import: %w", err)
		}

		if input.Pipelines != PipelineOptionConvert {
			return nil // assumes the value is enum.PipelineOptionIgnore
		}

		const convertPipelinesCommitMessage = "autoconvert pipeline"
		err = r.processPipelines(ctx, &systemPrincipal, repo, convertPipelinesCommitMessage)
		if err != nil {
			log.Warn().Err(err).Msg("failed to convert pipelines")
		}

		return nil
	}()
	if err != nil {
		log.Error().Err(err).Msg("failed repository import - cleanup git repository")

		repo.GitUID = gitUID // make sure to delete the correct directory

		if errDel := r.deleteGitRepository(context.Background(), &systemPrincipal, repo); errDel != nil {
			log.Warn().Err(errDel).
				Msg("failed to delete git repository after failed import")
		}

		return "", fmt.Errorf("failed to import repository: %w", err)
	}

	err = r.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypeRepositoryImportCompleted, repo)
	if err != nil {
		log.Warn().Err(err).Msg("failed to publish import completion SSE")
	}

	err = r.indexer.Index(ctx, repo)
	if err != nil {
		log.Warn().Err(err).Msg("failed to index repository")
	}

	log.Info().Msg("completed repository import")

	return "", nil
}

func (r *Repository) GetProgress(ctx context.Context, repo *types.Repository) (job.Progress, error) {
	progress, err := r.scheduler.GetJobProgress(ctx, JobIDFromRepoID(repo.ID))
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		if repo.State == enum.RepoStateGitImport {
			// if the job is not found but repo is marked as importing, return state=failed
			return job.FailProgress(), nil
		}

		// otherwise there either was no import, or it completed a long time ago (job cleaned up by now)
		return job.Progress{}, ErrNotFound
	}
	if err != nil {
		return job.Progress{}, fmt.Errorf("failed to get job progress: %w", err)
	}

	return progress, nil
}

func (r *Repository) Cancel(ctx context.Context, repo *types.Repository) error {
	if repo.State != enum.RepoStateGitImport {
		return nil
	}

	err := r.scheduler.CancelJob(ctx, JobIDFromRepoID(repo.ID))
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

	resp, err := r.git.CreateRepository(ctx, &git.CreateRepositoryParams{
		Actor: git.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		EnvVars:       envVars,
		DefaultBranch: r.defaultBranch,
		Files:         nil,
		Author: &git.Identity{
			Name:  principal.DisplayName,
			Email: principal.Email,
		},
		AuthorDate: &now,
		Committer: &git.Identity{
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

	syncOut, err := r.git.SyncRepository(ctx, &git.SyncRepositoryParams{
		WriteParams:       writeParams,
		Source:            sourceCloneURL,
		CreateIfNotExists: false,
		RefSpecs:          []string{"refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"},
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

	err = r.git.DeleteRepository(ctx, &git.DeleteRepositoryParams{
		WriteParams: writeParams,
	})
	if err != nil {
		return fmt.Errorf("failed to delete git repository: %w", err)
	}

	return nil
}

func (r *Repository) matchFiles(ctx context.Context,
	repo *types.Repository,
	ref string,
	dirPath string,
	pattern string,
	maxSize int,
) ([]pipelineFile, error) {
	resp, err := r.git.MatchFiles(ctx, &git.MatchFilesParams{
		ReadParams: git.ReadParams{RepoUID: repo.GitUID},
		Ref:        ref,
		DirPath:    dirPath,
		Pattern:    pattern,
		MaxSize:    maxSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert pipelines: %w", err)
	}

	pipelines := make([]pipelineFile, len(resp.Files))
	for i, pipeline := range resp.Files {
		pipelines[i] = pipelineFile{
			Name:          "",
			OriginalPath:  pipeline.Path,
			ConvertedPath: "",
			Content:       pipeline.Content,
		}
	}

	return pipelines, nil
}

func (r *Repository) createRPCWriteParams(ctx context.Context,
	principal *types.Principal,
	repo *types.Repository,
) (git.WriteParams, error) {
	envVars, err := r.createEnvVars(ctx, principal, repo.ID)
	if err != nil {
		return git.WriteParams{}, err
	}

	return git.WriteParams{
		Actor: git.Identity{
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
		r.urlProvider.GetInternalAPIURL(ctx),
		repoID,
		principal.ID,
		false,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return envVars, nil
}
