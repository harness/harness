// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

const (
	// maxGetContentFileSize specifies the maximum number of bytes a file content response contains.
	// If a file is any larger, the content is truncated.
	maxGetContentFileSize = 1 << 27 // 128 MB
)

type ContentType string

const (
	ContentTypeFile      ContentType = "file"
	ContentTypeDir       ContentType = "dir"
	ContentTypeSymlink   ContentType = "symlink"
	ContentTypeSubmodule ContentType = "submodule"
)

type ContentInfo struct {
	Type         ContentType `json:"type"`
	SHA          string      `json:"sha"`
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	LatestCommit *Commit     `json:"latest_commit,omitempty"`
}

type Commit struct {
	SHA       string    `json:"sha"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Author    Signature `json:"author"`
	Committer Signature `json:"committer"`
}

type Signature struct {
	Identity Identity  `json:"identity"`
	When     time.Time `json:"when"`
}

type Identity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GetContentOutput struct {
	ContentInfo
	Content Content `json:"content"`
}

type FileEncodingType string

const (
	FileEncodingTypeBase64 FileEncodingType = "base64"
)

// Content restricts the possible types of content returned by the api.
type Content interface {
	isContent()
}

type FileContent struct {
	Encoding FileEncodingType `json:"encoding"`
	Data     string           `json:"data"`
	Size     int64            `json:"size"`
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

/*
 * GetContent finds the content of the repo at the given path.
 * If no gitRef is provided, the content is retrieved from the default branch.
 * If includeLatestCommit is enabled, the response contains information of the latest commit that changed the object.
 */
func (c *Controller) GetContent(ctx context.Context, session *auth.Session, repoRef string,
	gitRef string, repoPath string, includeLatestCommit bool) (*GetContentOutput, error) {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, true); err != nil {
		return nil, err
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
		IncludeLatestCommit: includeLatestCommit,
	})
	if err != nil {
		return nil, err
	}

	info, err := mapToContentInfo(&treeNodeOutput.Node, treeNodeOutput.Commit)
	if err != nil {
		return nil, err
	}

	var content Content
	switch info.Type {
	case ContentTypeDir:
		// for getContent we don't want any recursiveness for dir content.
		content, err = c.getDirContent(ctx, readParams, gitRef, repoPath, includeLatestCommit, false)
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
		ContentInfo: *info,
		Content:     content,
	}, nil
}

func (c *Controller) getSubmoduleContent(ctx context.Context, readParams gitrpc.ReadParams, gitRef string,
	repoPath string, commitSHA string) (*SubmoduleContent, error) {
	output, err := c.gitRPCClient.GetSubmodule(ctx, &gitrpc.GetSubmoduleParams{
		ReadParams: readParams,
		GitREF:     gitRef,
		Path:       repoPath,
	})
	if err != nil {
		// TODO: handle not found error
		// This requires gitrpc to also return notfound though!
		return nil, fmt.Errorf("failed to get submodule: %w", err)
	}

	return &SubmoduleContent{
		URL:       output.Submodule.URL,
		CommitSHA: commitSHA,
	}, nil
}

func (c *Controller) getFileContent(ctx context.Context, readParams gitrpc.ReadParams,
	blobSHA string) (*FileContent, error) {
	output, err := c.gitRPCClient.GetBlob(ctx, &gitrpc.GetBlobParams{
		ReadParams: readParams,
		SHA:        blobSHA,
		SizeLimit:  maxGetContentFileSize,
	})
	if err != nil {
		// TODO: handle not found error
		// This requires gitrpc to also return notfound though!
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	return &FileContent{
		Size:     output.Blob.Size,
		Encoding: FileEncodingTypeBase64,
		Data:     base64.StdEncoding.EncodeToString(output.Blob.Content),
	}, nil
}

func (c *Controller) getSymlinkContent(ctx context.Context, readParams gitrpc.ReadParams,
	blobSHA string) (*SymlinkContent, error) {
	output, err := c.gitRPCClient.GetBlob(ctx, &gitrpc.GetBlobParams{
		ReadParams: readParams,
		SHA:        blobSHA,
		SizeLimit:  maxGetContentFileSize,
	})
	if err != nil {
		// TODO: handle not found error
		// This requires gitrpc to also return notfound though!
		return nil, fmt.Errorf("failed to get symlink: %w", err)
	}

	return &SymlinkContent{
		Size:   output.Blob.Size,
		Target: string(output.Blob.Content),
	}, nil
}

func (c *Controller) getDirContent(ctx context.Context, readParams gitrpc.ReadParams, gitRef string,
	repoPath string, includeLatestCommit bool, recursive bool) (*DirContent, error) {
	output, err := c.gitRPCClient.ListTreeNodes(ctx, &gitrpc.ListTreeNodeParams{
		ReadParams:          readParams,
		GitREF:              gitRef,
		Path:                repoPath,
		IncludeLatestCommit: includeLatestCommit,
		Recursive:           recursive,
	})
	if err != nil {
		// TODO: handle not found error
		// This requires gitrpc to also return notfound though!
		return nil, fmt.Errorf("failed to get content of dir: %w", err)
	}

	entries := make([]ContentInfo, len(output.Nodes))
	for i := range output.Nodes {
		node := output.Nodes[i]

		var entry *ContentInfo
		entry, err = mapToContentInfo(&node.TreeNode, node.Commit)
		if err != nil {
			return nil, err
		}
		entries[i] = *entry
	}

	return &DirContent{
		Entries: entries,
	}, nil
}

func mapToContentInfo(node *gitrpc.TreeNode, commit *gitrpc.Commit) (*ContentInfo, error) {
	// node data is expected
	if node == nil {
		return nil, fmt.Errorf("node can't be nil")
	}
	typ, err := mapNodeModeToContentType(node.Mode)
	if err != nil {
		return nil, err
	}

	res := &ContentInfo{
		Type: typ,
		SHA:  node.SHA,
		Name: node.Name,
		Path: node.Path,
	}

	// parse commit only if available
	if commit != nil {
		res.LatestCommit, err = mapCommit(commit)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func mapCommit(c *gitrpc.Commit) (*Commit, error) {
	if c == nil {
		return nil, fmt.Errorf("commit is nil")
	}

	author, err := mapSignature(&c.Author)
	if err != nil {
		return nil, fmt.Errorf("failed to map author: %w", err)
	}

	committer, err := mapSignature(&c.Committer)
	if err != nil {
		return nil, fmt.Errorf("failed to map committer: %w", err)
	}

	return &Commit{
		SHA:       c.SHA,
		Title:     c.Title,
		Message:   c.Message,
		Author:    *author,
		Committer: *committer,
	}, nil
}

func mapSignature(s *gitrpc.Signature) (*Signature, error) {
	if s == nil {
		return nil, fmt.Errorf("signature is nil")
	}

	return &Signature{
		Identity: Identity{
			Name:  s.Identity.Name,
			Email: s.Identity.Email,
		},
		When: s.When,
	}, nil
}

func mapNodeModeToContentType(m gitrpc.TreeNodeMode) (ContentType, error) {
	switch m {
	case gitrpc.TreeNodeModeFile, gitrpc.TreeNodeModeExec:
		return ContentTypeFile, nil
	case gitrpc.TreeNodeModeSymlink:
		return ContentTypeSymlink, nil
	case gitrpc.TreeNodeModeCommit:
		return ContentTypeSubmodule, nil
	case gitrpc.TreeNodeModeTree:
		return ContentTypeDir, nil
	default:
		return ContentTypeFile, fmt.Errorf("unsupported tree node mode '%s'", m)
	}
}
