// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"context"
	"fmt"

	"github.com/harness/gitness/build/triggerer"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/go-scm/scm"
)

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	branch string,
) (*types.Execution, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path,
		pipelineUID, enum.PermissionPipelineExecute)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	// If the branch is empty, use the default branch specified in the pipeline.
	if branch == "" {
		branch = pipeline.DefaultBranch
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
		Author:      commit.Author.Identity.Name,
		AuthorName:  commit.Author.Identity.Name,
		AuthorEmail: commit.Author.Identity.Email,
		Ref:         ref,
		Message:     commit.Message,
		Title:       "", // we expect this to be empty.
		Before:      commit.SHA,
		After:       commit.SHA,
		Sender:      session.Principal.UID,
		Source:      branch,
		Target:      branch,
		Action:      types.EventCustom,
		Params:      map[string]string{},
		Timestamp:   commit.Author.When.UnixMilli(),
	}

	// Trigger the execution
	return c.triggerer.Trigger(ctx, pipeline, hook)
}
