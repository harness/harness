// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/harness/gitness/internal/gitrpc/rpc"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// TODO: this should be configurable
	FileTransferChunkSize = 1024
)

var ErrNoParamsProvided = errors.New("no params provided")

type File struct {
	Path    string
	Content []byte
}

type CreateRepositoryParams struct {
	DefaultBranch string
	Files         []File
}

type CreateRepositoryOutput struct {
	UID string
}

type GetTreeNodeParams struct {
	// RepoUID is the uid of the git repository
	RepoUID string
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF              string
	Path                string
	IncludeLatestCommit bool
}

type GetTreeNodeOutput struct {
	Node   TreeNode
	Commit *Commit
}

type GetBlobParams struct {
	RepoUID   string
	SHA       string
	SizeLimit int64
}

type GetBlobOutput struct {
	Blob Blob
}

type Blob struct {
	SHA  string
	Size int64
	// Content contains the data of the blob
	// NOTE: can be only partial data - compare len(.content) with .size
	Content []byte
}

type GetSubmoduleParams struct {
	// RepoUID is the uid of the git repository
	RepoUID string
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF string
	Path   string
}

type GetSubmoduleOutput struct {
	Submodule Submodule
}
type Submodule struct {
	Name string
	URL  string
}

type ListTreeNodeParams struct {
	// RepoUID is the uid of the git repository
	RepoUID string
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF              string
	Path                string
	IncludeLatestCommit bool
	Recursive           bool
}

type ListTreeNodeOutput struct {
	Nodes []TreeNodeWithCommit
}

type TreeNodeWithCommit struct {
	TreeNode
	Commit *Commit
}

type ListCommitsParams struct {
	// RepoUID is the uid of the git repository
	RepoUID string
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF   string
	Page     int32
	PageSize int32
}

type ListCommitsOutput struct {
	TotalCount int64
	Commits    []Commit
}

type ListBranchesParams struct {
	// RepoUID is the uid of the git repository
	RepoUID       string
	IncludeCommit bool
	Page          int32
	PageSize      int32
}

type ListBranchesOutput struct {
	TotalCount int64
	Branches   []Branch
}

type Branch struct {
	Name   string
	Commit *Commit
}

type TreeNode struct {
	Type TreeNodeType
	Mode TreeNodeMode
	SHA  string
	Name string
	Path string
}

// TreeNodeType specifies the different types of nodes in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeType (proto).
type TreeNodeType string

const (
	TreeNodeTypeTree   TreeNodeType = "tree"
	TreeNodeTypeBlob   TreeNodeType = "blob"
	TreeNodeTypeCommit TreeNodeType = "commit"
)

// TreeNodeMode specifies the different modes of a node in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeMode (proto).
type TreeNodeMode string

const (
	TreeNodeModeFile    TreeNodeMode = "file"
	TreeNodeModeSymlink TreeNodeMode = "symlink"
	TreeNodeModeExec    TreeNodeMode = "exec"
	TreeNodeModeTree    TreeNodeMode = "tree"
	TreeNodeModeCommit  TreeNodeMode = "commit"
)

type Commit struct {
	SHA       string
	Title     string
	Message   string
	Author    Signature
	Committer Signature
}

type Signature struct {
	Identity Identity
	When     time.Time
}

type Identity struct {
	Name  string
	Email string
}

type Client struct {
	conn        *grpc.ClientConn
	repoService rpc.RepositoryServiceClient
}

func InitClient(remoteAddr string) (*Client, error) {
	conn, err := grpc.Dial(remoteAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:        conn,
		repoService: rpc.NewRepositoryServiceClient(conn),
	}, nil
}

func newRepositoryUID() (string, error) {
	return gonanoid.New()
}

func (c *Client) CreateRepository(ctx context.Context,
	params *CreateRepositoryParams) (*CreateRepositoryOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	uid, err := newRepositoryUID()
	if err != nil {
		return nil, fmt.Errorf("failed to create new uid: %w", err)
	}
	log.Ctx(ctx).Info().
		Msgf("Create new git repository with uid '%s' and default branch '%s'", uid, params.DefaultBranch)

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	stream, err := c.repoService.CreateRepository(ctx)
	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Info().Msgf("Send header")

	req := &rpc.CreateRepositoryRequest{
		Data: &rpc.CreateRepositoryRequest_Header{
			Header: &rpc.CreateRepositoryRequestHeader{
				Uid:           uid,
				DefaultBranch: params.DefaultBranch,
			},
		},
	}
	if err = stream.Send(req); err != nil {
		return nil, err
	}

	for _, file := range params.Files {
		log.Ctx(ctx).Info().Msgf("Send file %s", file.Path)

		err = uploadFile(file, FileTransferChunkSize, func(fs *rpc.FileUpload) error {
			return stream.Send(&rpc.CreateRepositoryRequest{
				Data: &rpc.CreateRepositoryRequest_File{
					File: fs,
				},
			})
		})
		if err != nil {
			return nil, err
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Info().Msgf("completed git repo setup.")

	return &CreateRepositoryOutput{UID: uid}, nil
}

func uploadFile(
	file File,
	chunkSize int,
	send func(*rpc.FileUpload) error,
) error {
	log.Info().Msgf("start sending %v", file.Path)

	// send filename message
	header := &rpc.FileUpload{
		Data: &rpc.FileUpload_Header{
			Header: &rpc.FileUploadHeader{
				Path: file.Path,
			},
		},
	}
	if err := send(header); err != nil {
		return fmt.Errorf("failed to send file upload header: %w", err)
	}

	err := sendChunks(file.Content, chunkSize, func(c *rpc.Chunk) error {
		return send(&rpc.FileUpload{
			Data: &rpc.FileUpload_Chunk{
				Chunk: c,
			},
		})
	})
	if err != nil {
		return fmt.Errorf("failed to send file data: %w", err)
	}

	log.Info().Msgf("completed sending %v", file.Path)

	return nil
}

func sendChunks(
	content []byte,
	chunkSize int,
	send func(*rpc.Chunk) error) error {
	buffer := make([]byte, chunkSize)
	reader := bytes.NewReader(content)

	for {
		n, err := reader.Read(buffer)
		if errors.Is(err, io.EOF) {
			err = send(&rpc.Chunk{
				Eof:  true,
				Data: buffer[:n],
			})
			if err != nil {
				return err
			}

			break
		}
		if err != nil {
			return fmt.Errorf("cannot read buffer: %w", err)
		}

		err = send(&rpc.Chunk{
			Eof:  false,
			Data: buffer[:n],
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) GetTreeNode(ctx context.Context, params *GetTreeNodeParams) (*GetTreeNodeOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	resp, err := c.repoService.GetTreeNode(ctx, &rpc.GetTreeNodeRequest{
		RepoUid:             params.RepoUID,
		GitRef:              params.GitREF,
		Path:                params.Path,
		IncludeLatestCommit: params.IncludeLatestCommit,
	})
	if err != nil {
		return nil, err
	}

	node, err := mapRPCTreeNode(resp.GetNode())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc node: %w", err)
	}
	var commit *Commit
	if resp.GetCommit() != nil {
		commit, err = mapRPCCommit(resp.GetCommit())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc commit: %w", err)
		}
	}

	return &GetTreeNodeOutput{
		Node:   node,
		Commit: commit,
	}, nil
}

func (c *Client) GetSubmodule(ctx context.Context, params *GetSubmoduleParams) (*GetSubmoduleOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	resp, err := c.repoService.GetSubmodule(ctx, &rpc.GetSubmoduleRequest{
		RepoUid: params.RepoUID,
		GitRef:  params.GitREF,
		Path:    params.Path,
	})
	if err != nil {
		return nil, err
	}
	if resp.GetSubmodule() == nil {
		return nil, fmt.Errorf("rpc submodule is nil")
	}

	return &GetSubmoduleOutput{
		Submodule: Submodule{
			Name: resp.GetSubmodule().Name,
			URL:  resp.GetSubmodule().Url,
		},
	}, nil
}

func (c *Client) GetBlob(ctx context.Context, params *GetBlobParams) (*GetBlobOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	resp, err := c.repoService.GetBlob(ctx, &rpc.GetBlobRequest{
		RepoUid:   params.RepoUID,
		Sha:       params.SHA,
		SizeLimit: params.SizeLimit,
	})
	if err != nil {
		return nil, err
	}

	blob := resp.GetBlob()
	if blob == nil {
		return nil, fmt.Errorf("rpc blob is nil")
	}

	return &GetBlobOutput{
		Blob: Blob{
			SHA:     blob.GetSha(),
			Size:    blob.GetSize(),
			Content: blob.GetContent(),
		},
	}, nil
}

func (c *Client) ListTreeNodes(ctx context.Context, params *ListTreeNodeParams) (*ListTreeNodeOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	stream, err := c.repoService.ListTreeNodes(ctx, &rpc.ListTreeNodesRequest{
		RepoUid:             params.RepoUID,
		GitRef:              params.GitREF,
		Path:                params.Path,
		IncludeLatestCommit: params.IncludeLatestCommit,
		Recursive:           params.Recursive,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for tree nodes: %w", err)
	}

	nodes := make([]TreeNodeWithCommit, 0, 16)
	for {
		var next *rpc.ListTreeNodesResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, fmt.Errorf("received unexpected error from rpc: %w", err)
		}

		var node TreeNode
		node, err = mapRPCTreeNode(next.GetNode())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc node: %w", err)
		}
		var commit *Commit
		if next.GetCommit() != nil {
			commit, err = mapRPCCommit(next.GetCommit())
			if err != nil {
				return nil, fmt.Errorf("failed to map rpc commit: %w", err)
			}
		}

		nodes = append(nodes, TreeNodeWithCommit{
			TreeNode: node,
			Commit:   commit,
		})
	}

	// TODO: is this needed?
	err = stream.CloseSend()
	if err != nil {
		return nil, fmt.Errorf("failed to close stream")
	}

	return &ListTreeNodeOutput{
		Nodes: nodes,
	}, nil
}

func (c *Client) ListCommits(ctx context.Context, params *ListCommitsParams) (*ListCommitsOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	stream, err := c.repoService.ListCommits(ctx, &rpc.ListCommitsRequest{
		RepoUid:  params.RepoUID,
		GitRef:   params.GitREF,
		Page:     params.Page,
		PageSize: params.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for commits: %w", err)
	}

	// get header first
	header, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("error occured while receiving header: %w", err)
	}
	if header.GetHeader() == nil {
		return nil, fmt.Errorf("header missing")
	}
	output := &ListCommitsOutput{
		TotalCount: header.GetHeader().TotalCount,
		Commits:    make([]Commit, 0, params.PageSize),
	}

	for {
		var next *rpc.ListCommitsResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, fmt.Errorf("received unexpected error from rpc: %w", err)
		}
		if next.GetCommit() == nil {
			return nil, fmt.Errorf("expected commit message")
		}

		var commit *Commit
		commit, err = mapRPCCommit(next.GetCommit())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc commit: %w", err)
		}

		output.Commits = append(output.Commits, *commit)
	}

	// TODO: is this needed?
	err = stream.CloseSend()
	if err != nil {
		return nil, fmt.Errorf("failed to close stream")
	}

	return output, nil
}

func (c *Client) ListBranches(ctx context.Context, params *ListBranchesParams) (*ListBranchesOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	stream, err := c.repoService.ListBranches(ctx, &rpc.ListBranchesRequest{
		RepoUid:       params.RepoUID,
		IncludeCommit: params.IncludeCommit,
		Page:          params.Page,
		PageSize:      params.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for branches: %w", err)
	}

	// get header first
	header, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("error occured while receiving header: %w", err)
	}
	if header.GetHeader() == nil {
		return nil, fmt.Errorf("header missing")
	}
	output := &ListBranchesOutput{
		TotalCount: header.GetHeader().TotalCount,
		Branches:   make([]Branch, 0, params.PageSize),
	}

	for {
		var next *rpc.ListBranchesResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, fmt.Errorf("received unexpected error from rpc: %w", err)
		}
		if next.GetBranch() == nil {
			return nil, fmt.Errorf("expected branch message")
		}

		var branch *Branch
		branch, err = mapRPCBranch(next.GetBranch())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc branch: %w", err)
		}

		output.Branches = append(output.Branches, *branch)
	}

	// TODO: is this needed?
	err = stream.CloseSend()
	if err != nil {
		return nil, fmt.Errorf("failed to close stream")
	}

	return output, nil
}

func mapRPCBranch(b *rpc.Branch) (*Branch, error) {
	if b == nil {
		return nil, fmt.Errorf("rpc branch is nil")
	}

	var commit *Commit
	if b.GetCommit() != nil {
		var err error
		commit, err = mapRPCCommit(b.GetCommit())
		if err != nil {
			return nil, err
		}
	}

	return &Branch{
		Name:   b.Name,
		Commit: commit,
	}, nil
}

func mapRPCCommit(c *rpc.Commit) (*Commit, error) {
	if c == nil {
		return nil, fmt.Errorf("rpc commit is nil")
	}

	author, err := mapRPCSignature(c.GetAuthor())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc author: %w", err)
	}

	comitter, err := mapRPCSignature(c.GetCommitter())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc committer: %w", err)
	}

	return &Commit{
		SHA:       c.GetSha(),
		Title:     c.GetTitle(),
		Message:   c.GetMessage(),
		Author:    author,
		Committer: comitter,
	}, nil
}

func mapRPCSignature(s *rpc.Signature) (Signature, error) {
	if s == nil {
		return Signature{}, fmt.Errorf("rpc signature is nil")
	}

	identity, err := mapRPCIdentity(s.GetIdentity())
	if err != nil {
		return Signature{}, fmt.Errorf("failed to map rpc identity: %w", err)
	}

	when := time.Unix(s.When, 0)

	return Signature{
		Identity: identity,
		When:     when,
	}, nil
}

func mapRPCIdentity(id *rpc.Identity) (Identity, error) {
	if id == nil {
		return Identity{}, fmt.Errorf("rpc identity is nil")
	}

	return Identity{
		Name:  id.GetName(),
		Email: id.GetEmail(),
	}, nil
}

func mapRPCTreeNode(n *rpc.TreeNode) (TreeNode, error) {
	if n == nil {
		return TreeNode{}, fmt.Errorf("rpc tree node is nil")
	}

	nodeType, err := mapRPCTreeNodeType(n.GetType())
	if err != nil {
		return TreeNode{}, err
	}

	mode, err := mapRPCTreeNodeMode(n.GetMode())
	if err != nil {
		return TreeNode{}, err
	}

	return TreeNode{
		Type: nodeType,
		Mode: mode,
		SHA:  n.GetSha(),
		Name: n.GetName(),
		Path: n.GetPath(),
	}, nil
}

func mapRPCTreeNodeType(t rpc.TreeNodeType) (TreeNodeType, error) {
	switch t {
	case rpc.TreeNodeType_TreeNodeTypeBlob:
		return TreeNodeTypeBlob, nil
	case rpc.TreeNodeType_TreeNodeTypeCommit:
		return TreeNodeTypeCommit, nil
	case rpc.TreeNodeType_TreeNodeTypeTree:
		return TreeNodeTypeTree, nil
	default:
		return TreeNodeTypeBlob, fmt.Errorf("unkown rpc tree node type: %d", t)
	}
}

func mapRPCTreeNodeMode(m rpc.TreeNodeMode) (TreeNodeMode, error) {
	switch m {
	case rpc.TreeNodeMode_TreeNodeModeFile:
		return TreeNodeModeFile, nil
	case rpc.TreeNodeMode_TreeNodeModeExec:
		return TreeNodeModeExec, nil
	case rpc.TreeNodeMode_TreeNodeModeSymlink:
		return TreeNodeModeSymlink, nil
	case rpc.TreeNodeMode_TreeNodeModeCommit:
		return TreeNodeModeCommit, nil
	case rpc.TreeNodeMode_TreeNodeModeTree:
		return TreeNodeModeTree, nil
	default:
		return TreeNodeModeFile, fmt.Errorf("unkown rpc tree node mode: %d", m)
	}
}
