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
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/go-convert/convert/bitbucket"
	"github.com/drone/go-convert/convert/circle"
	"github.com/drone/go-convert/convert/drone"
	"github.com/drone/go-convert/convert/github"
	"github.com/drone/go-convert/convert/gitlab"
	"github.com/rs/zerolog/log"
)

type Importer struct {
	defaultBranch string
	urlProvider   gitnessurl.Provider
	git           git.Interface
	tx            dbtx.Transactor
	repoStore     store.RepoStore
	pipelineStore store.PipelineStore
	triggerStore  store.TriggerStore
	repoFinder    refcache.RepoFinder
	sseStreamer   sse.Streamer
	indexer       keywordsearch.Indexer
	publicAccess  publicaccess.Service
	eventReporter *repoevents.Reporter
	auditService  audit.Service
	settings      *settings.Service
}

func NewImporter(
	defaultBranch string,
	urlProvider gitnessurl.Provider,
	git git.Interface,
	tx dbtx.Transactor,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	triggerStore store.TriggerStore,
	repoFinder refcache.RepoFinder,
	sseStreamer sse.Streamer,
	indexer keywordsearch.Indexer,
	publicAccess publicaccess.Service,
	eventReporter *repoevents.Reporter,
	auditService audit.Service,
	settings *settings.Service,
) *Importer {
	return &Importer{
		defaultBranch: defaultBranch,
		urlProvider:   urlProvider,
		git:           git,
		tx:            tx,
		repoStore:     repoStore,
		pipelineStore: pipelineStore,
		triggerStore:  triggerStore,
		repoFinder:    repoFinder,
		sseStreamer:   sseStreamer,
		indexer:       indexer,
		publicAccess:  publicAccess,
		eventReporter: eventReporter,
		auditService:  auditService,
		settings:      settings,
	}
}

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

func (r *Importer) Import(ctx context.Context, input Input) error {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal

	if input.CloneURL == "" {
		return errors.InvalidArgument("missing git repository clone URL")
	}

	repoURL, err := url.Parse(input.CloneURL)
	if err != nil {
		return fmt.Errorf("failed to parse git clone URL: %w", err)
	}

	repoURL.User = url.UserPassword(input.GitUser, input.GitPass)
	cloneURLWithAuth := repoURL.String()

	repo, err := r.repoStore.Find(ctx, input.RepoID)
	if err != nil {
		return fmt.Errorf("failed to find repo by id: %w", err)
	}

	if repo.State != enum.RepoStateGitImport {
		return errors.InvalidArgumentf("repository %s is not being imported", repo.Identifier)
	}

	log := log.Ctx(ctx).With().
		Int64("repo.id", repo.ID).
		Str("repo.path", repo.Path).
		Logger()

	log.Info().Msg("configure access mode")

	parentPath, _, err := paths.DisectLeaf(repo.Path)
	if err != nil {
		return fmt.Errorf("failed to disect path %q: %w", repo.Path, err)
	}
	isPublicAccessSupported, err := r.publicAccess.IsPublicAccessSupported(ctx, enum.PublicResourceTypeRepo, parentPath)
	if err != nil {
		return fmt.Errorf(
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
		return fmt.Errorf("failed to set repo access mode: %w", err)
	}

	if isRepoPublic {
		err = r.auditService.Log(ctx,
			bootstrap.NewSystemServiceSession().Principal,
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

	// revert this when import fetches LFS objects
	if err := r.settings.RepoSet(ctx, repo.ID, settings.KeyGitLFSEnabled, false); err != nil {
		log.Warn().Err(err).Msg("failed to disable Git LFS in repository settings")
	}

	log.Info().Msg("create git repository")

	gitUID, err := r.createGitRepository(ctx, &systemPrincipal, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to create empty git repository: %w", err)
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

		r.repoFinder.MarkChanged(ctx, repo.Core())

		log.Info().Msg("sync repository")

		err = r.syncGitRepository(ctx, &systemPrincipal, repo, cloneURLWithAuth)
		if err != nil {
			return fmt.Errorf("failed to sync git repository from '%s': %w", input.CloneURL, err)
		}

		log.Info().Msgf("successfully synced repository (with default branch: %q)", repo.DefaultBranch)

		log.Info().Msg("update repo in DB")

		repo, err = r.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
			if repo.State != enum.RepoStateGitImport {
				return errors.New("repository has already finished importing")
			}

			repo.GitUID = gitUID
			repo.State = enum.RepoStateActive

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update repository after import: %w", err)
		}

		r.repoFinder.MarkChanged(ctx, repo.Core())

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

		if errDel := r.deleteGitRepository(context.WithoutCancel(ctx), &systemPrincipal, repo); errDel != nil {
			log.Warn().Err(errDel).
				Msg("failed to delete git repository after failed import")
		}

		return fmt.Errorf("failed to import repository: %w", err)
	}

	r.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypeRepositoryImportCompleted, repo)

	r.eventReporter.Created(ctx, &repoevents.CreatedPayload{
		Base: repoevents.Base{
			RepoID:      repo.ID,
			PrincipalID: bootstrap.NewSystemServiceSession().Principal.ID,
		},
		IsPublic:     input.Public,
		ImportedFrom: input.CloneURL,
	})

	err = r.indexer.Index(ctx, repo)
	if err != nil {
		log.Warn().Err(err).Msg("failed to index repository")
	}

	log.Info().Msg("completed repository import")

	return nil
}

func (r *Importer) createGitRepository(
	ctx context.Context,
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

func (r *Importer) syncGitRepository(
	ctx context.Context,
	principal *types.Principal,
	repo *types.Repository,
	sourceCloneURL string,
) error {
	writeParams, err := r.createRPCWriteParams(ctx, principal, repo)
	if err != nil {
		return err
	}

	_, err = r.git.SyncRepository(ctx, &git.SyncRepositoryParams{
		WriteParams:       writeParams,
		Source:            sourceCloneURL,
		CreateIfNotExists: false,
		RefSpecs:          []string{"refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"},
		DefaultBranch:     repo.DefaultBranch,
	})
	if err != nil {
		return fmt.Errorf("failed to sync repository: %w", err)
	}

	return nil
}

func (r *Importer) deleteGitRepository(
	ctx context.Context,
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
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete git repository: %w", err)
	}

	return nil
}

func (r *Importer) matchFiles(
	ctx context.Context,
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

func (r *Importer) createRPCWriteParams(
	ctx context.Context,
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

func (r *Importer) createEnvVars(
	ctx context.Context,
	principal *types.Principal,
	repoID int64,
) (map[string]string, error) {
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		r.urlProvider.GetInternalAPIURL(ctx),
		repoID,
		principal.ID,
		true,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return envVars, nil
}

type pipelineFile struct {
	Name          string
	OriginalPath  string
	ConvertedPath string
	Content       []byte
}

func (r *Importer) processPipelines(ctx context.Context,
	principal *types.Principal,
	repo *types.Repository,
	commitMessage string,
) error {
	writeParams, err := r.createRPCWriteParams(ctx, principal, repo)
	if err != nil {
		return err
	}

	pipelineFiles := r.convertPipelines(ctx, repo)
	if len(pipelineFiles) == 0 {
		return nil
	}

	actions := make([]git.CommitFileAction, len(pipelineFiles))
	for i, file := range pipelineFiles {
		actions[i] = git.CommitFileAction{
			Action:  git.CreateAction,
			Path:    file.ConvertedPath,
			Payload: file.Content,
			SHA:     sha.None,
		}
	}

	now := time.Now()
	identity := &git.Identity{
		Name:  principal.DisplayName,
		Email: principal.Email,
	}

	_, err = r.git.CommitFiles(ctx, &git.CommitFilesParams{
		WriteParams:   writeParams,
		Message:       commitMessage,
		Branch:        repo.DefaultBranch,
		NewBranch:     repo.DefaultBranch,
		Actions:       actions,
		Committer:     identity,
		CommitterDate: &now,
		Author:        identity,
		AuthorDate:    &now,
	})
	if err != nil {
		return fmt.Errorf("failed to commit converted pipeline files: %w", err)
	}

	nowMilli := now.UnixMilli()

	err = r.tx.WithTx(ctx, func(ctx context.Context) error {
		for _, p := range pipelineFiles {
			pipeline := &types.Pipeline{
				Description:   "",
				RepoID:        repo.ID,
				Identifier:    p.Name,
				CreatedBy:     principal.ID,
				Seq:           0,
				DefaultBranch: repo.DefaultBranch,
				ConfigPath:    p.ConvertedPath,
				Created:       nowMilli,
				Updated:       nowMilli,
				Version:       0,
			}

			err = r.pipelineStore.Create(ctx, pipeline)
			if err != nil {
				return fmt.Errorf("pipeline creation failed: %w", err)
			}

			// Try to create a default trigger on pipeline creation.
			// Default trigger operations are set on pull request created, reopened or updated.
			// We log an error on failure but don't fail the op.
			trigger := &types.Trigger{
				Description: "auto-created trigger on pipeline conversion",
				Created:     nowMilli,
				Updated:     nowMilli,
				PipelineID:  pipeline.ID,
				RepoID:      pipeline.RepoID,
				CreatedBy:   principal.ID,
				Identifier:  "default",
				Actions: []enum.TriggerAction{enum.TriggerActionPullReqCreated,
					enum.TriggerActionPullReqReopened, enum.TriggerActionPullReqBranchUpdated},
				Disabled: false,
				Version:  0,
			}
			err = r.triggerStore.Create(ctx, trigger)
			if err != nil {
				return fmt.Errorf("failed to create auto trigger on pipeline creation: %w", err)
			}
		}

		return nil
	}, dbtx.TxDefault)
	if err != nil {
		return fmt.Errorf("failed to insert pipelines and triggers: %w", err)
	}

	return nil
}

// convertPipelines converts pipelines found in the repository.
// Note: For GitHub actions, there can be multiple.
func (r *Importer) convertPipelines(ctx context.Context,
	repo *types.Repository,
) []pipelineFile {
	const maxSize = 65536

	match := func(dirPath, regExpDef string) []pipelineFile {
		files, err := r.matchFiles(ctx, repo, repo.DefaultBranch, dirPath, regExpDef, maxSize)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to find pipeline file(s) '%s' in '%s'",
				regExpDef, dirPath)
			return nil
		}
		return files
	}

	if files := match("", ".drone.yml"); len(files) > 0 {
		converted := convertPipelineFiles(ctx, files, func() pipelineConverter { return drone.New() })
		if len(converted) > 0 {
			return converted
		}
	}

	if files := match("", "bitbucket-pipelines.yml"); len(files) > 0 {
		converted := convertPipelineFiles(ctx, files, func() pipelineConverter { return bitbucket.New() })
		if len(converted) > 0 {
			return converted
		}
	}

	if files := match("", ".gitlab-ci.yml"); len(files) > 0 {
		converted := convertPipelineFiles(ctx, files, func() pipelineConverter { return gitlab.New() })
		if len(converted) > 0 {
			return converted
		}
	}

	if files := match(".circleci", "config.yml"); len(files) > 0 {
		converted := convertPipelineFiles(ctx, files, func() pipelineConverter { return circle.New() })
		if len(converted) > 0 {
			return converted
		}
	}

	filesYML := match(".github/workflows", "*.yml")
	filesYAML := match(".github/workflows", "*.yaml")
	//nolint:gocritic // intended usage
	files := append(filesYML, filesYAML...)
	converted := convertPipelineFiles(ctx, files, func() pipelineConverter { return github.New() })
	if len(converted) > 0 {
		return converted
	}

	return nil
}

type pipelineConverter interface {
	ConvertBytes([]byte) ([]byte, error)
}

func convertPipelineFiles(ctx context.Context,
	files []pipelineFile,
	gen func() pipelineConverter,
) []pipelineFile {
	const (
		harnessPipelineName     = "pipeline"
		harnessPipelineNameOnly = "default-" + harnessPipelineName
		harnessPipelineDir      = ".harness"
		harnessPipelineFileOnly = harnessPipelineDir + "/pipeline.yaml"
	)

	result := make([]pipelineFile, 0, len(files))
	for _, file := range files {
		data, err := gen().ConvertBytes(file.Content)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to convert pipeline file %s", file.OriginalPath)
			continue
		}

		var pipelineName string
		var pipelinePath string

		if len(files) == 1 {
			pipelineName = harnessPipelineNameOnly
			pipelinePath = harnessPipelineFileOnly
		} else {
			base := path.Base(file.OriginalPath)
			base = strings.TrimSuffix(base, path.Ext(base))
			pipelineName = harnessPipelineName + "-" + base
			pipelinePath = harnessPipelineDir + "/" + base + ".yaml"
		}

		result = append(result, pipelineFile{
			Name:          pipelineName,
			OriginalPath:  file.OriginalPath,
			ConvertedPath: pipelinePath,
			Content:       data,
		})
	}

	return result
}
