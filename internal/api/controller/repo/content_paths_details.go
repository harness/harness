// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

type PathsDetailsInput struct {
	Paths []string `json:"paths"`
}

type PathsDetailsOutput struct {
	Details []gitrpc.PathDetails `json:"details"`
}

// PathsDetails finds the additional info about the provided paths of the repo.
// If no gitRef is provided, the content is retrieved from the default branch.
func (c *Controller) PathsDetails(ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
	input PathsDetailsInput,
) (PathsDetailsOutput, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return PathsDetailsOutput{}, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, true); err != nil {
		return PathsDetailsOutput{}, err
	}

	if len(input.Paths) == 0 {
		return PathsDetailsOutput{}, nil
	}

	if len(input.Paths) > 50 {
		return PathsDetailsOutput{}, usererror.BadRequest("maximum number of elements in the Paths array is 25")
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	// create read params once
	readParams := CreateRPCReadParams(repo)

	result, err := c.gitRPCClient.PathsDetails(ctx, gitrpc.PathsDetailsParams{
		ReadParams: readParams,
		GitREF:     gitRef,
		Paths:      input.Paths,
	})
	if err != nil {
		return PathsDetailsOutput{}, err
	}

	return PathsDetailsOutput{
		Details: result.Details,
	}, nil
}
