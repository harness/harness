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
	"path/filepath"
	"time"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO: this should be taken as a struct input defined in proto.
func (s RepositoryService) AddFilesAndPush(
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
		return processGitErrorf(err, "failed to add files")
	}
	now := time.Now()
	err = s.adapter.Commit(repoPath, types.CommitChangesOptions{
		// TODO: Add gitness signature
		Committer: types.Signature{
			Identity: types.Identity{
				Name:  committer.Name,
				Email: committer.Email,
			},
			When: now,
		},
		// TODO: Add gitness signature
		Author: types.Signature{
			Identity: types.Identity{
				Name:  author.Name,
				Email: author.Email,
			},
			When: now,
		},
		Message: message,
	})
	if err != nil {
		return processGitErrorf(err, "failed to commit files")
	}
	err = s.adapter.Push(ctx, repoPath, types.PushOptions{
		// TODO: Don't hard-code
		Remote:  remote,
		Branch:  branch,
		Force:   false,
		Mirror:  false,
		Env:     nil,
		Timeout: 0,
	})
	if err != nil {
		return processGitErrorf(err, "failed to push files")
	}

	return nil
}

func (s RepositoryService) handleFileUploadIfAvailable(ctx context.Context, basePath string,
	nextFSElement func() (*rpc.FileUpload, error)) (string, error) {
	log := log.Ctx(ctx)

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
