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

package repo

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types/enum"
)

// Raw finds the file of the repo at the given path and returns its raw content.
// If no gitRef is provided, the content is retrieved from the default branch.
func (c *Controller) Raw(ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
	path string,
) (io.ReadCloser, int64, sha.SHA, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, sha.Nil, err
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	// create read params once
	readParams := git.CreateReadParams(repo)
	treeNodeOutput, err := c.git.GetTreeNode(ctx, &git.GetTreeNodeParams{
		ReadParams:          readParams,
		GitREF:              gitRef,
		Path:                path,
		IncludeLatestCommit: false,
	})
	if err != nil {
		return nil, 0, sha.Nil, fmt.Errorf("failed to read tree node: %w", err)
	}

	// viewing Raw content is only supported for blob content
	if treeNodeOutput.Node.Type != git.TreeNodeTypeBlob {
		return nil, 0, sha.Nil, usererror.BadRequestf(
			"Object in '%s' at '/%s' is of type '%s'. Only objects of type %s support raw viewing.",
			gitRef, path, treeNodeOutput.Node.Type, git.TreeNodeTypeBlob)
	}

	blobReader, err := c.git.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: readParams,
		SHA:        treeNodeOutput.Node.SHA,
		SizeLimit:  0, // no size limit, we stream whatever data there is
	})
	if err != nil {
		return nil, 0, sha.Nil, fmt.Errorf("failed to read blob: %w", err)
	}

	return blobReader.Content, blobReader.ContentSize, blobReader.SHA, nil
}
