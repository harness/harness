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
	"path"
	"strings"
	"time"

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

type pipelineFile struct {
	Name          string
	OriginalPath  string
	ConvertedPath string
	Content       []byte
}

func (r *Repository) processPipelines(ctx context.Context,
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
func (r *Repository) convertPipelines(ctx context.Context,
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
