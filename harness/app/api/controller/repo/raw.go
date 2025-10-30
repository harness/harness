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
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type RawContent struct {
	Data io.ReadCloser
	Size int64
	SHA  sha.SHA
}

// Raw finds the file of the repo at the given path and returns its raw content.
// If no gitRef is provided, the content is retrieved from the default branch.
func (c *Controller) Raw(ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
	path string,
) (*RawContent, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to read tree node: %w", err)
	}

	// viewing Raw content is only supported for blob content
	if treeNodeOutput.Node.Type != git.TreeNodeTypeBlob {
		return nil, usererror.BadRequestf(
			"Object in '%s' at '/%s' is of type '%s'. Only objects of type %s support raw viewing.",
			gitRef, path, treeNodeOutput.Node.Type, git.TreeNodeTypeBlob)
	}

	blobReader, err := c.git.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: readParams,
		SHA:        treeNodeOutput.Node.SHA,
		SizeLimit:  0, // no size limit, we stream whatever data there is
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read blob: %w", err)
	}

	gitLFSEnabled, err := settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyGitLFSEnabled,
		settings.DefaultGitLFSEnabled,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check settings for Git LFS enabled: %w", err)
	}

	if !gitLFSEnabled {
		return &RawContent{
			Data: blobReader.Content,
			Size: blobReader.ContentSize,
			SHA:  blobReader.SHA,
		}, nil
	}

	// check if blob is an LFS pointer
	headerContent, err := io.ReadAll(io.LimitReader(blobReader.Content, parser.LfsPointerMaxSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	lfsInfo, ok := parser.IsLFSPointer(ctx, headerContent, blobReader.Size)
	if ok {
		lfsContent, err := c.lfsCtrl.DownloadNoAuth(ctx, repo.ID, lfsInfo.OID)
		if err == nil {
			return &RawContent{
				Data: lfsContent,
				Size: lfsInfo.Size,
				SHA:  blobReader.SHA,
			}, nil
		}
		if !errors.Is(err, gitness_store.ErrResourceNotFound) {
			return nil, fmt.Errorf("failed to download LFS file: %w", err)
		}
	}

	return &RawContent{
		Data: &types.MultiReadCloser{
			Reader:    io.MultiReader(bytes.NewBuffer(headerContent), blobReader.Content),
			CloseFunc: blobReader.Content.Close,
		},
		Size: blobReader.ContentSize,
		SHA:  blobReader.SHA,
	}, nil
}
