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

package aiagent

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) GetAnalysis(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	executionNum int64,
) (*types.AnalyseExecutionOutput, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, usererror.BadRequestf("failed to find repo %s", repoRef)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineIdentifier, enum.PermissionPipelineView)
	if err != nil {
		return nil, usererror.Forbidden(fmt.Sprintf("not allowed to view pipeline %s", pipelineIdentifier))
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, pipelineIdentifier)
	if err != nil {
		return nil, usererror.BadRequestf("failed to find pipeline: %s", pipelineIdentifier)
	}

	execution, err := c.executionStore.FindByNumber(ctx, pipeline.ID, executionNum)
	if err != nil {
		return nil, usererror.BadRequestf("failed to find execution %d", executionNum)
	}

	if execution.Status == enum.CIStatusSuccess {
		return nil, usererror.BadRequestf("execution %d is not a failed execution", executionNum)
	}

	// ToDo: put actual values
	payload := &CommitPayload{}
	branch := ""

	_, err = c.commit(ctx, session, repo, payload)
	if err != nil {
		return &types.AnalyseExecutionOutput{}, err
	}

	return &types.AnalyseExecutionOutput{Branch: branch, Summary: ""}, nil
}

type CommitPayload struct {
	Title     string
	Message   string
	Branch    string
	NewBranch string
	Files     []*Files
}

type Files struct {
	action  git.FileAction
	path    string
	content string
	SHA     sha.SHA
}

func (c *Controller) commit(ctx context.Context,
	session *auth.Session,
	repo *types.Repository,
	payload *CommitPayload) (types.CommitFilesResponse, error) {
	files := payload.Files
	actions := make([]git.CommitFileAction, len(files))
	for i, file := range files {
		rawPayload := []byte(file.content)
		actions[i] = git.CommitFileAction{
			Action:  file.action,
			Path:    file.path,
			Payload: rawPayload,
			SHA:     file.SHA,
		}
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return types.CommitFilesResponse{}, fmt.Errorf("failed to create RPC write params: %w", err)
	}
	now := time.Now()
	commit, err := c.git.CommitFiles(ctx, &git.CommitFilesParams{
		WriteParams:   writeParams,
		Title:         payload.Title,
		Message:       payload.Message,
		Branch:        payload.Branch,
		NewBranch:     payload.NewBranch,
		Actions:       actions,
		Committer:     identityFromPrincipal(bootstrap.NewSystemServiceSession().Principal),
		CommitterDate: &now,
		Author:        identityFromPrincipal(session.Principal),
		AuthorDate:    &now,
	})
	if err != nil {
		return types.CommitFilesResponse{}, err
	}

	return types.CommitFilesResponse{
		CommitID: commit.CommitID.String(),
	}, nil
}

func identityFromPrincipal(p types.Principal) *git.Identity {
	return &git.Identity{
		Name:  p.DisplayName,
		Email: p.Email,
	}
}
