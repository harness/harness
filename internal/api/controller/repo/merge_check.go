// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

type MergeCheck struct {
	Mergeable     bool     `json:"mergeable"`
	ConflictFiles []string `json:"conflict_files,omitempty"`
}

func (c *Controller) MergeCheck(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	diffPath string,
) (MergeCheck, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return MergeCheck{}, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return MergeCheck{}, fmt.Errorf("access check failed: %w", err)
	}

	info, err := parseDiffPath(diffPath)
	if err != nil {
		return MergeCheck{}, err
	}

	writeParams, err := CreateRPCWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return MergeCheck{}, fmt.Errorf("failed to create rpc write params: %w", err)
	}

	_, err = c.gitRPCClient.Merge(ctx, &gitrpc.MergeParams{
		WriteParams: writeParams,
		BaseBranch:  info.BaseRef,
		HeadRepoUID: writeParams.RepoUID, // forks are not supported for now
		HeadBranch:  info.HeadRef,
	})
	if err != nil {
		if gitrpc.ErrorStatus(err) == gitrpc.StatusNotMergeable {
			return MergeCheck{
				Mergeable:     false,
				ConflictFiles: gitrpc.AsConflictFilesError(err),
			}, nil
		}
		return MergeCheck{}, fmt.Errorf("merge check execution failed: %w", err)
	}

	return MergeCheck{
		Mergeable: true,
	}, nil
}
