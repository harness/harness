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
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/harness/gitness/internal/gitrpc/rpc"
)

const (
	maxFileSize    = 1 << 20
	repoSubdirName = "repos"
	gitRepoSuffix  = "git"
)

var (
	// TODO: Should be matching the sytem identity from config.
	SystemIdentity = &rpc.Identity{
		Name:  "gitness",
		Email: "system@gitness",
	}
)

type repositoryService struct {
	rpc.UnimplementedRepositoryServiceServer
	adapter   gitAdapter
	store     *localStore
	reposRoot string
}

func newRepositoryService(adapter gitAdapter, store *localStore, gitRoot string) (*repositoryService, error) {
	reposRoot := filepath.Join(gitRoot, repoSubdirName)
	if _, err := os.Stat(reposRoot); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(reposRoot, 0o700); err != nil {
			return nil, err
		}
	}

	return &repositoryService{
		adapter:   adapter,
		store:     store,
		reposRoot: reposRoot,
	}, nil
}

func (s repositoryService) getFullPathForRepo(uid string) string {
	return filepath.Join(s.reposRoot, fmt.Sprintf("%s.%s", uid, gitRepoSuffix))
}

//nolint:gocognit // need to refactor this code
func (s repositoryService) CreateRepository(stream rpc.RepositoryService_CreateRepositoryServer) error {
	// first get repo params from stream
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "cannot receive create repository data")
	}

	header := req.GetHeader()
	if header == nil {
		return status.Errorf(codes.Internal, "expected header to be first message in stream")
	}
	log.Info().Msgf("received a create repository request %v", header)

	repoPath := s.getFullPathForRepo(header.GetUid())
	if _, err = os.Stat(repoPath); !os.IsNotExist(err) {
		return status.Errorf(codes.AlreadyExists, "repository exists already: %v", repoPath)
	}

	// create repository in repos folder
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = s.adapter.InitRepository(ctx, repoPath, true)
	if err != nil {
		// on error cleanup repo dir
		defer func(path string) {
			_ = os.RemoveAll(path)
		}(repoPath)
		return fmt.Errorf("CreateRepository error: %w", err)
	}

	// update default branch (currently set to non-existent branch)
	err = s.adapter.SetDefaultBranch(ctx, repoPath, header.GetDefaultBranch(), true)
	if err != nil {
		return fmt.Errorf("error updating default branch for repo %s: %w", header.GetUid(), err)
	}

	// we need temp dir for cloning
	tempDir, err := os.MkdirTemp("", "*-"+header.GetUid())
	if err != nil {
		return fmt.Errorf("error creating temp dir for repo %s: %w", header.GetUid(), err)
	}
	defer func(path string) {
		// when repo is successfully created remove temp dir
		err2 := os.RemoveAll(path)
		if err2 != nil {
			log.Err(err2).Msg("failed to cleanup temporary dir.")
		}
	}(tempDir)

	// Clone repository to temp dir
	if err = s.adapter.Clone(ctx, repoPath, tempDir, cloneRepoOption{}); err != nil {
		return status.Errorf(codes.Internal, "failed to clone repo: %v", err)
	}

	// logic for receiving files
	filePaths := make([]string, 0, 16)
	for {
		var filePath string
		filePath, err = s.handleFileUploadIfAvailable(tempDir, func() (*rpc.FileUpload, error) {
			m, err2 := stream.Recv()
			if err2 != nil {
				return nil, err2
			}
			return m.GetFile(), nil
		})
		if errors.Is(err, io.EOF) {
			log.Info().Msg("received stream EOF")
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive file: %v", err)
		}

		filePaths = append(filePaths, filePath)
	}

	if len(filePaths) > 0 {
		// NOTE: This creates the branch in origin repo (as it doesn't exist as of now)
		// TODO: this should at least be a constant and not hardcoded?
		if err = s.AddFilesAndPush(ctx, tempDir, filePaths, "HEAD:"+header.GetDefaultBranch(), SystemIdentity, SystemIdentity,
			"origin", "initial commit"); err != nil {
			return err
		}
	}

	res := &rpc.CreateRepositoryResponse{}
	err = stream.SendAndClose(res)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot send completion response: %v", err)
	}

	log.Info().Msgf("repository created. Path: %s", repoPath)
	return nil
}

func (s repositoryService) handleFileUploadIfAvailable(basePath string,
	nextFSElement func() (*rpc.FileUpload, error)) (string, error) {
	log.Info().Msg("waiting to receive file upload header")
	header, err := getFileStreamHeader(nextFSElement)
	if err != nil {
		return "", err
	}

	log.Info().Msgf("storing file at %s", header.Path)
	// work with file content chunks
	fileData := bytes.Buffer{}
	fileSize := 0
	for {
		log.Debug().Msg("waiting to receive data")

		var chunk *rpc.Chunk
		chunk, err = getFileUploadChunk(nextFSElement)
		if errors.Is(err, io.EOF) {
			// only for a header we expect a stream EOF error (for chunk its a chunk.EOF).
			return "", fmt.Errorf("data stream ended unexpectedly")
		}
		if err != nil {
			return "", err
		}

		size := len(chunk.Data)

		if size > 0 {
			log.Debug().Msgf("received a chunk with size: %d", size)

			// TODO: file size could be checked on client side?
			fileSize += size
			if fileSize > maxFileSize {
				return "", status.Errorf(codes.InvalidArgument, "file is too large: %d > %d", fileSize, maxFileSize)
			}

			// TODO: write in file as we go (instead of in buffer)
			_, err = fileData.Write(chunk.Data)
			if err != nil {
				return "", status.Errorf(codes.Internal, "cannot write chunk data: %v", err)
			}
		}

		if chunk.Eof {
			log.Info().Msg("Received file EOF")
			break
		}
	}
	fullPath := filepath.Join(basePath, header.Path)
	log.Info().Msgf("saving file at path %s", fullPath)
	_, err = s.store.Save(fullPath, fileData)
	if err != nil {
		return "", status.Errorf(codes.Internal, "cannot save file to the store: %v", err)
	}

	return fullPath, nil
}

func getFileStreamHeader(nextFileUpload func() (*rpc.FileUpload, error)) (*rpc.FileUploadHeader, error) {
	fs, err := getFileUpload(nextFileUpload)
	if err != nil {
		return nil, err
	}

	header := fs.GetHeader()
	if header == nil {
		return nil, status.Errorf(codes.Internal, "file stream is in wrong order - expected header")
	}

	return header, nil
}

func getFileUploadChunk(nextFileUpload func() (*rpc.FileUpload, error)) (*rpc.Chunk, error) {
	fs, err := getFileUpload(nextFileUpload)
	if err != nil {
		return nil, err
	}

	chunk := fs.GetChunk()
	if chunk == nil {
		return nil, status.Errorf(codes.Internal, "file stream is in wrong order - expected chunk")
	}

	return chunk, nil
}

func getFileUpload(nextFileUpload func() (*rpc.FileUpload, error)) (*rpc.FileUpload, error) {
	fs, err := nextFileUpload()
	if err != nil {
		return nil, err
	}
	if fs == nil {
		return nil, status.Errorf(codes.Internal, "file stream wasn't found")
	}
	return fs, nil
}

// TODO: this should be taken as a struct input defined in proto.
func (s repositoryService) AddFilesAndPush(
	ctx context.Context,
	repoPath string,
	filePaths []string,
	branch string,
	author *rpc.Identity,
	committer *rpc.Identity,
	remote string,
	message string,
) error {
	if author == nil || committer == nil {
		return status.Errorf(codes.InvalidArgument, "both author and committer have to be provided")
	}

	err := s.adapter.AddFiles(repoPath, false, filePaths...)
	if err != nil {
		return err
	}
	now := time.Now()
	err = s.adapter.Commit(repoPath, commitChangesOptions{
		// TODO: Add gitness signature
		committer: signature{
			identity: identity{
				name:  committer.Name,
				email: committer.Email,
			},
			when: now,
		},
		// TODO: Add gitness signature
		author: signature{
			identity: identity{
				name:  author.Name,
				email: author.Email,
			},
			when: now,
		},
		message: message,
	})
	if err != nil {
		return err
	}
	err = s.adapter.Push(ctx, repoPath, pushOptions{
		// TODO: Don't hard-code
		remote:  remote,
		branch:  branch,
		force:   false,
		mirror:  false,
		env:     nil,
		timeout: 0,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s repositoryService) GetTreeNode(ctx context.Context,
	request *rpc.GetTreeNodeRequest) (*rpc.GetTreeNodeResponse, error) {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitNode, err := s.adapter.GetTreeNode(ctx, repoPath, request.GetGitRef(), request.GetPath())
	if err != nil {
		return nil, err
	}

	res := &rpc.GetTreeNodeResponse{
		Node: &rpc.TreeNode{
			Type: mapGitNodeType(gitNode.nodeType),
			Mode: mapGitMode(gitNode.mode),
			Sha:  gitNode.sha,
			Name: gitNode.name,
			Path: gitNode.path,
		},
	}

	// TODO: improve performance, could be done in lower layer?
	if request.GetIncludeLatestCommit() {
		var commit *rpc.Commit
		commit, err = s.getLatestCommit(ctx, repoPath, request.GetGitRef(), request.GetPath())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get latest commit: %v", err)
		}
		res.Commit = commit
	}

	return res, nil
}

func (s repositoryService) GetSubmodule(ctx context.Context,
	request *rpc.GetSubmoduleRequest) (*rpc.GetSubmoduleResponse, error) {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitSubmodule, err := s.adapter.GetSubmodule(ctx, repoPath, request.GetGitRef(), request.GetPath())
	if err != nil {
		return nil, err
	}

	return &rpc.GetSubmoduleResponse{
		Submodule: &rpc.Submodule{
			Name: gitSubmodule.name,
			Url:  gitSubmodule.url,
		},
	}, nil
}

func (s repositoryService) GetBlob(ctx context.Context, request *rpc.GetBlobRequest) (*rpc.GetBlobResponse, error) {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitBlob, err := s.adapter.GetBlob(ctx, repoPath, request.GetSha(), request.GetSizeLimit())
	if err != nil {
		return nil, err
	}

	return &rpc.GetBlobResponse{
		Blob: &rpc.Blob{
			Sha:     request.GetSha(),
			Size:    gitBlob.size,
			Content: gitBlob.content,
		},
	}, nil
}

func (s repositoryService) ListTreeNodes(request *rpc.ListTreeNodesRequest,
	stream rpc.RepositoryService_ListTreeNodesServer) error {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())

	gitNodes, err := s.adapter.ListTreeNodes(stream.Context(), repoPath,
		request.GetGitRef(), request.GetPath(), request.GetRecursive(), request.GetIncludeLatestCommit())
	if err != nil {
		return status.Errorf(codes.Internal, "failed to list nodes: %v", err)
	}

	log.Trace().Msgf("git adapter returned %d nodes", len(gitNodes))

	for _, gitNode := range gitNodes {
		var commit *rpc.Commit
		if request.GetIncludeLatestCommit() {
			commit, err = mapGitCommit(gitNode.commit)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
			}
		}

		err = stream.Send(&rpc.ListTreeNodesResponse{
			Node: &rpc.TreeNode{
				Type: mapGitNodeType(gitNode.nodeType),
				Mode: mapGitMode(gitNode.mode),
				Sha:  gitNode.sha,
				Name: gitNode.name,
				Path: gitNode.path,
			},
			Commit: commit,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send node: %v", err)
		}
	}

	return nil
}

func (s repositoryService) ListCommits(request *rpc.ListCommitsRequest,
	stream rpc.RepositoryService_ListCommitsServer) error {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())

	gitCommits, totalCount, err := s.adapter.ListCommits(stream.Context(), repoPath, request.GetGitRef(),
		int(request.GetPage()), int(request.GetPageSize()))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to list commits: %v", err)
	}

	log.Trace().Msgf("git adapter returned %d commits (total: %d)", len(gitCommits), totalCount)

	// send info about total number of commits first
	err = stream.Send(&rpc.ListCommitsResponse{
		Data: &rpc.ListCommitsResponse_Header{
			Header: &rpc.ListCommitsResponseHeader{
				TotalCount: totalCount,
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send response header: %v", err)
	}

	for i := range gitCommits {
		var commit *rpc.Commit
		commit, err = mapGitCommit(&gitCommits[i])
		if err != nil {
			return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
		}

		err = stream.Send(&rpc.ListCommitsResponse{
			Data: &rpc.ListCommitsResponse_Commit{
				Commit: commit,
			},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send commit: %v", err)
		}
	}

	return nil
}

func (s repositoryService) ListBranches(request *rpc.ListBranchesRequest,
	stream rpc.RepositoryService_ListBranchesServer) error {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())

	gitBranches, totalCount, err := s.adapter.ListBranches(stream.Context(), repoPath,
		request.GetIncludeCommit(), int(request.GetPage()), int(request.GetPageSize()))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to list branches: %v", err)
	}

	log.Trace().Msgf("git adapter returned %d branches (total: %d)", len(gitBranches), totalCount)

	// send info about total number of branches first
	err = stream.Send(&rpc.ListBranchesResponse{
		Data: &rpc.ListBranchesResponse_Header{
			Header: &rpc.ListBranchesResponseHeader{
				TotalCount: totalCount,
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send response header: %v", err)
	}

	for i := range gitBranches {
		var branch *rpc.Branch
		branch, err = mapGitBranch(&gitBranches[i])
		if err != nil {
			return status.Errorf(codes.Internal, "failed to map git branch: %v", err)
		}

		err = stream.Send(&rpc.ListBranchesResponse{
			Data: &rpc.ListBranchesResponse_Branch{
				Branch: branch,
			},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send branch: %v", err)
		}
	}

	return nil
}

func (s repositoryService) getLatestCommit(ctx context.Context, repoPath string,
	ref string, path string) (*rpc.Commit, error) {
	gitCommit, err := s.adapter.GetLatestCommit(ctx, repoPath, ref, path)
	if err != nil {
		return nil, err
	}

	return mapGitCommit(gitCommit)
}

// TODO: Add UTs to ensure enum values match!
func mapGitNodeType(t treeNodeType) rpc.TreeNodeType {
	return rpc.TreeNodeType(t)
}

// TODO: Add UTs to ensure enum values match!
func mapGitMode(m treeNodeMode) rpc.TreeNodeMode {
	return rpc.TreeNodeMode(m)
}

func mapGitBranch(gitBranch *branch) (*rpc.Branch, error) {
	if gitBranch == nil {
		return nil, status.Errorf(codes.Internal, "git branch is nil")
	}

	var commit *rpc.Commit
	if gitBranch.commit != nil {
		var err error
		commit, err = mapGitCommit(gitBranch.commit)
		if err != nil {
			return nil, err
		}
	}

	return &rpc.Branch{
		Name:   gitBranch.name,
		Commit: commit,
	}, nil
}

func mapGitCommit(gitCommit *commit) (*rpc.Commit, error) {
	if gitCommit == nil {
		return nil, status.Errorf(codes.Internal, "git commit is nil")
	}

	return &rpc.Commit{
		Sha:     gitCommit.sha,
		Title:   gitCommit.title,
		Message: gitCommit.message,
		Author: &rpc.Signature{
			Identity: &rpc.Identity{
				Name:  gitCommit.author.identity.name,
				Email: gitCommit.author.identity.email,
			},
			When: gitCommit.author.when.Unix(),
		},
		Committer: &rpc.Signature{
			Identity: &rpc.Identity{
				Name:  gitCommit.committer.identity.name,
				Email: gitCommit.committer.identity.email,
			},
			When: gitCommit.committer.when.Unix(),
		},
	}, nil
}
