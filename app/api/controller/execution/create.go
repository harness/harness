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

package execution

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/go-scm/scm"
)

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	branch string,
) (*types.Execution, error) {
	repo, err := c.getRepoCheckPipelineAccess(ctx, session, repoRef, pipelineIdentifier, enum.PermissionPipelineExecute)
	if err != nil {
		return nil, err
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, pipelineIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	// If the branch is empty, use the default branch specified in the pipeline.
	// It that is also empty, use the repo default branch.
	if branch == "" {
		branch = pipeline.DefaultBranch
		if branch == "" {
			branch = repo.DefaultBranch
		}
	}
	// expand the branch to a git reference.
	ref := scm.ExpandRef(branch, "refs/heads")

	// Fetch the commit information from the commits service.
	commit, err := c.commitService.FindRef(ctx, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commit: %w", err)
	}

	// Create manual hook for execution.
	hook := &triggerer.Hook{
		Trigger:     session.Principal.UID, // who/what triggered the build, different from commit author
		AuthorLogin: commit.Author.Identity.Name,
		TriggeredBy: session.Principal.ID,
		AuthorName:  commit.Author.Identity.Name,
		AuthorEmail: commit.Author.Identity.Email,
		Ref:         ref,
		Message:     commit.Message,
		Title:       commit.Title,
		Before:      commit.SHA.String(),
		After:       commit.SHA.String(),
		Sender:      session.Principal.UID,
		Source:      branch,
		Target:      branch,
		Params:      map[string]string{},
		Timestamp:   commit.Author.When.UnixMilli(),
	}

	// Trigger the execution
	return c.triggerer.Trigger(ctx, pipeline, hook)
}
