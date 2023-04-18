// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

/*
 * Raw finds the file of the repo at the given path and returns its raw content.
 * If no gitRef is provided, the content is retrieved from the default branch.
 */
func (c *Controller) Raw(ctx context.Context, session *auth.Session, repoRef string,
	gitRef string, repoPath string) (io.Reader, int64, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, true); err != nil {
		return nil, 0, err
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	// create read params once
	readParams := CreateRPCReadParams(repo)
	treeNodeOutput, err := c.gitRPCClient.GetTreeNode(ctx, &gitrpc.GetTreeNodeParams{
		ReadParams:          readParams,
		GitREF:              gitRef,
		Path:                repoPath,
		IncludeLatestCommit: false,
	})
	if err != nil {
		return nil, 0, err
	}

	// viewing Raw content is only supported for blob content
	if treeNodeOutput.Node.Type != gitrpc.TreeNodeTypeBlob {
		return nil, 0, usererror.BadRequestf(
			"Object in '%s' at '/%s' is of type '%s'. Only objects of type %s support raw viewing.",
			gitRef, repoPath, treeNodeOutput.Node.Type, gitrpc.TreeNodeTypeBlob)
	}

	blobReader, err := c.gitRPCClient.GetBlob(ctx, &gitrpc.GetBlobParams{
		ReadParams: readParams,
		SHA:        treeNodeOutput.Node.SHA,
		SizeLimit:  0, // no size limit, we stream whatever data there is
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read blob from gitrpc: %w", err)
	}

	return blobReader.Content, blobReader.ContentSize, nil
}
