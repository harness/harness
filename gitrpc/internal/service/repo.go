// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxFileSize   = 1 << 20
	gitRepoSuffix = "git"

	gitReferenceNamePrefixBranch = "refs/heads/"
	gitReferenceNamePrefixTag    = "refs/tags/"
)

var (
	// TODO: Should be matching the sytem identity from config.
	SystemIdentity = &rpc.Identity{
		Name:  "gitness",
		Email: "system@gitness",
	}
)

type Storage interface {
	Save(filePath string, data bytes.Buffer) (string, error)
}

type RepositoryService struct {
	rpc.UnimplementedRepositoryServiceServer
	adapter   GitAdapter
	store     Storage
	reposRoot string
}

func NewRepositoryService(adapter GitAdapter, store Storage, reposRoot string) (*RepositoryService, error) {
	return &RepositoryService{
		adapter:   adapter,
		store:     store,
		reposRoot: reposRoot,
	}, nil
}

//nolint:gocognit // need to refactor this code
func (s RepositoryService) CreateRepository(stream rpc.RepositoryService_CreateRepositoryServer) error {
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

	repoPath := getFullPathForRepo(s.reposRoot, header.GetUid())
	if _, err = os.Stat(repoPath); !os.IsNotExist(err) {
		return status.Errorf(codes.AlreadyExists, "repository exists already: %v", repoPath)
	}

	// create repository in repos folder
	ctx, cancel := context.WithCancel(context.Background()) // todo: there is stream.Context()
	defer cancel()
	err = s.adapter.InitRepository(ctx, repoPath, true)
	if err != nil {
		// on error cleanup repo dir
		if errCleanup := os.RemoveAll(repoPath); errCleanup != nil {
			log.Err(errCleanup).Msg("failed to cleanup repository dir")
		}
		return processGitErrorf(err, "failed to initialize the repository")
	}

	// update default branch (currently set to non-existent branch)
	err = s.adapter.SetDefaultBranch(ctx, repoPath, header.GetDefaultBranch(), true)
	if err != nil {
		return processGitErrorf(err, "error updating default branch for repo '%s'", header.GetUid())
	}

	// we need temp dir for cloning
	tempDir, err := os.MkdirTemp("", "*-"+header.GetUid())
	if err != nil {
		return fmt.Errorf("error creating temp dir for repo %s: %w", header.GetUid(), err)
	}
	defer func(path string) {
		// when repo is successfully created remove temp dir
		errRm := os.RemoveAll(path)
		if errRm != nil {
			log.Err(errRm).Msg("failed to cleanup temporary dir.")
		}
	}(tempDir)

	// Clone repository to temp dir
	if err = s.adapter.Clone(ctx, repoPath, tempDir, types.CloneRepoOptions{}); err != nil {
		return processGitErrorf(err, "failed to clone repo")
	}

	// logic for receiving files
	filePaths := make([]string, 0, 16)
	for {
		var filePath string
		filePath, err = s.handleFileUploadIfAvailable(tempDir, func() (*rpc.FileUpload, error) {
			m, errStream := stream.Recv()
			if errStream != nil {
				return nil, errStream
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
