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

type repositoryService struct {
	rpc.UnimplementedRepositoryServiceServer
	adapter gitAdapter
	store   UploadStore
}

//nolint:funlen,gocognit // needs to refactor this code
func (s repositoryService) CreateRepository(stream rpc.RepositoryService_CreateRepositoryServer) error {
	repoRoot := getRepoRoot()
	// first get repo params from stream
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot receive create repository data")
	}

	repo := req.GetRepository()
	log.Info().Msgf("receive an create repository request %v", repo)
	targetPath := filepath.Join(repoRoot, repo.Name)
	if _, err = os.Stat(targetPath); !os.IsNotExist(err) {
		return status.Errorf(codes.AlreadyExists, "repository exists already: %v", targetPath)
	}

	// create repository in repos folder
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = s.adapter.InitRepository(ctx, targetPath, true)
	if err != nil {
		// on error cleanup repo dir
		defer func(path string) {
			_ = os.RemoveAll(path)
		}(targetPath)
		return fmt.Errorf("CreateRepository error: %w", err)
	}

	// we need temp dir for cloning
	tempDir, err := os.MkdirTemp("", "*-"+repo.Name)
	if err != nil {
		return fmt.Errorf("error creating temp dir for repo %s: %w", repo.Name, err)
	}
	defer func(path string) {
		// when repo is successfully created remove temp dir
		_ = os.RemoveAll(path)
	}(tempDir)

	// Clone repository to temp dir
	if err = s.adapter.Clone(ctx, targetPath, tempDir, cloneRepoOption{}); err != nil {
		return err
	}

	// logic for receiving the files
	files := make([]string, 0, 16)
	for {
		log.Info().Msg("waiting to receive file path")
		req, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Info().Msg("EOF found")
			break
		}
		// get file path message
		filePath := req.GetFilepath()
		log.Info().Msgf("receiving file %s", filePath)
		// work with file content chunks
		fileData := bytes.Buffer{}
		fileSize := 0
		for {
			log.Debug().Msg("waiting to receive data")

			req, err = stream.Recv()
			if errors.Is(err, io.EOF) {
				return status.Errorf(codes.Internal, "received unexpected end of stream: %v", err)
			}

			chunk := req.GetChunk()
			size := len(chunk.Data)

			if size > 0 {
				log.Debug().Msgf("received a chunk with size: %d", size)
				fileSize += size
				if fileSize > maxFileSize {
					return status.Errorf(codes.InvalidArgument, "file is too large: %d > %d", fileSize, maxFileSize)
				}
				_, err = fileData.Write(chunk.Data)
				if err != nil {
					return status.Errorf(codes.Internal, "cannot write chunk data: %v", err)
				}
			}

			if chunk.Eof {
				log.Info().Msg("Received file EOF")
				break
			}
		}
		log.Info().Msgf("saving file %s in repo path %s", filePath, tempDir)
		fullPath := filepath.Join(tempDir, filePath)
		_, err = s.store.Save(fullPath, fileData)
		if err != nil {
			return status.Errorf(codes.Internal, "cannot save file to the store: %v", err)
		}
		files = append(files, filePath)
	}
	res := &rpc.CreateRepositoryResponse{
		TempPath: tempDir,
	}

	if len(files) > 0 {
		if _, err = s.AddFilesAndPush(ctx, &rpc.AddFilesAndPushRequest{
			RepoPath: tempDir,
			Message:  "initial commit",
			Files:    files,
		}); err != nil {
			return err
		}
	}

	err = stream.SendAndClose(res)
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot send completion response: %v", err)
	}

	log.Info().Msgf("repository created: %s, path: %s", repo.Name, targetPath)
	return nil
}

func (s repositoryService) AddFilesAndPush(
	ctx context.Context,
	params *rpc.AddFilesAndPushRequest,
) (*rpc.AddFilesAndPushResponse, error) {
	err := s.adapter.AddFiles(params.GetRepoPath(), false, params.GetFiles()...)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	err = s.adapter.Commit(params.GetRepoPath(), commitChangesOptions{
		// TODO: Add gitness signature
		committer: &signature{
			name:  "enver",
			email: "enver.bisevac@harness.io",
			when:  now,
		},
		// TODO: Add gitness signature
		author: &signature{
			name:  "enver",
			email: "enver.bisevac@harness.io",
			when:  now,
		},
		message: params.GetMessage(),
	})
	if err != nil {
		return nil, err
	}
	err = s.adapter.Push(ctx, params.GetRepoPath(), pushOptions{
		// TODO: Don't hard-code
		remote:  "origin",
		branch:  "",
		force:   false,
		mirror:  false,
		env:     nil,
		timeout: 0,
	})
	if err != nil {
		return nil, err
	}
	// TODO: caller should delete repo if needed (as we didn't create it here)
	err = os.RemoveAll(params.GetRepoPath())
	if err != nil {
		return nil, err
	}
	return &rpc.AddFilesAndPushResponse{}, nil
}
