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
	"encoding/base64"
	"fmt"
	"io"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// maxGetContentFileSize specifies the maximum number of bytes a file content response contains.
	// If a file is any larger, the content is truncated.
	maxGetContentFileSize = 1 << 22 // 4 MB
)

type ContentType string

const (
	ContentTypeFile      ContentType = "file"
	ContentTypeDir       ContentType = "dir"
	ContentTypeSymlink   ContentType = "symlink"
	ContentTypeSubmodule ContentType = "submodule"
)

type ContentInfo struct {
	Type         ContentType   `json:"type"`
	SHA          string        `json:"sha"`
	Name         string        `json:"name"`
	Path         string        `json:"path"`
	LatestCommit *types.Commit `json:"latest_commit,omitempty"`
}

type GetContentOutput struct {
	ContentInfo
	Content Content `json:"content"`
}

// Content restricts the possible types of content returned by the api.
type Content interface {
	isContent()
}

type FileContent struct {
	Encoding enum.ContentEncodingType `json:"encoding"`
	Data     string                   `json:"data"`
	Size     int64                    `json:"size"`
	DataSize int64                    `json:"data_size"`
}

func (c *FileContent) isContent() {}

type SymlinkContent struct {
	Target string `json:"target"`
	Size   int64  `json:"size"`
}

func (c *SymlinkContent) isContent() {}

type DirContent struct {
	Entries []ContentInfo `json:"entries"`
}

func (c *DirContent) isContent() {}

type SubmoduleContent struct {
	URL       string `json:"url"`
	CommitSHA string `json:"commit_sha"`
}

func (c *SubmoduleContent) isContent() {}

// GetContent finds the content of the repo at the given path.
// If no gitRef is provided, the content is retrieved from the default branch.
func (c *Controller) GetContent(ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
	repoPath string,
	includeLatestCommit bool,
) (*GetContentOutput, error) {
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
		Path:                repoPath,
		IncludeLatestCommit: includeLatestCommit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read tree node: %w", err)
	}

	info, err := mapToContentInfo(treeNodeOutput.Node, treeNodeOutput.Commit, includeLatestCommit)
	if err != nil {
		return nil, err
	}

	var content Content
	switch info.Type {
	case ContentTypeDir:
		content, err = c.getDirContent(ctx, readParams, gitRef, repoPath, includeLatestCommit)
	case ContentTypeFile:
		content, err = c.getFileContent(ctx, readParams, info.SHA)
	case ContentTypeSymlink:
		content, err = c.getSymlinkContent(ctx, readParams, info.SHA)
	case ContentTypeSubmodule:
		content, err = c.getSubmoduleContent(ctx, readParams, gitRef, repoPath, info.SHA)
	default:
		err = fmt.Errorf("unknown tree node type '%s'", treeNodeOutput.Node.Type)
	}

	if err != nil {
		return nil, err
	}

	return &GetContentOutput{
		ContentInfo: info,
		Content:     content,
	}, nil
}

func (c *Controller) getSubmoduleContent(ctx context.Context,
	readParams git.ReadParams,
	gitRef string,
	repoPath string,
	commitSHA string,
) (*SubmoduleContent, error) {
	output, err := c.git.GetSubmodule(ctx, &git.GetSubmoduleParams{
		ReadParams: readParams,
		GitREF:     gitRef,
		Path:       repoPath,
	})
	if err != nil {
		// TODO: handle not found error
		return nil, fmt.Errorf("failed to get submodule: %w", err)
	}

	return &SubmoduleContent{
		URL:       output.Submodule.URL,
		CommitSHA: commitSHA,
	}, nil
}

func (c *Controller) getFileContent(ctx context.Context,
	readParams git.ReadParams,
	blobSHA string,
) (*FileContent, error) {
	output, err := c.git.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: readParams,
		SHA:        blobSHA,
		SizeLimit:  maxGetContentFileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	defer func() {
		if err := output.Content.Close(); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to close blob content reader.")
		}
	}()

	content, err := io.ReadAll(output.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob content: %w", err)
	}

	return &FileContent{
		Size:     output.Size,
		DataSize: output.ContentSize,
		Encoding: enum.ContentEncodingTypeBase64,
		Data:     base64.StdEncoding.EncodeToString(content),
	}, nil
}

func (c *Controller) getSymlinkContent(ctx context.Context,
	readParams git.ReadParams,
	blobSHA string,
) (*SymlinkContent, error) {
	output, err := c.git.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: readParams,
		SHA:        blobSHA,
		SizeLimit:  maxGetContentFileSize, // TODO: do we need to guard against too big symlinks?
	})
	if err != nil {
		// TODO: handle not found error
		return nil, fmt.Errorf("failed to get symlink: %w", err)
	}

	defer func() {
		if err := output.Content.Close(); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to close blob content reader.")
		}
	}()

	content, err := io.ReadAll(output.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob content: %w", err)
	}

	return &SymlinkContent{
		Size:   output.Size,
		Target: string(content),
	}, nil
}

func (c *Controller) getDirContent(ctx context.Context,
	readParams git.ReadParams,
	gitRef string,
	repoPath string,
	includeLatestCommit bool,
) (*DirContent, error) {
	output, err := c.git.ListTreeNodes(ctx, &git.ListTreeNodeParams{
		ReadParams:          readParams,
		GitREF:              gitRef,
		Path:                repoPath,
		IncludeLatestCommit: includeLatestCommit,
	})
	if err != nil {
		// TODO: handle not found error
		return nil, fmt.Errorf("failed to get content of dir: %w", err)
	}

	entries := make([]ContentInfo, len(output.Nodes))
	for i, node := range output.Nodes {
		entries[i], err = mapToContentInfo(node, nil, false)
		if err != nil {
			return nil, err
		}
	}

	return &DirContent{
		Entries: entries,
	}, nil
}

func mapToContentInfo(node git.TreeNode, commit *git.Commit, includeLatestCommit bool) (ContentInfo, error) {
	typ, err := mapNodeModeToContentType(node.Mode)
	if err != nil {
		return ContentInfo{}, err
	}

	res := ContentInfo{
		Type: typ,
		SHA:  node.SHA,
		Name: node.Name,
		Path: node.Path,
	}

	// parse commit only if available
	if commit != nil && includeLatestCommit {
		res.LatestCommit, err = controller.MapCommit(commit)
		if err != nil {
			return ContentInfo{}, err
		}
	}

	return res, nil
}

func mapNodeModeToContentType(m git.TreeNodeMode) (ContentType, error) {
	switch m {
	case git.TreeNodeModeFile, git.TreeNodeModeExec:
		return ContentTypeFile, nil
	case git.TreeNodeModeSymlink:
		return ContentTypeSymlink, nil
	case git.TreeNodeModeCommit:
		return ContentTypeSubmodule, nil
	case git.TreeNodeModeTree:
		return ContentTypeDir, nil
	default:
		return ContentTypeFile, errors.Internal(nil, "unsupported tree node mode '%s'", m)
	}
}
