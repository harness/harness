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
func (s RepositoryService) addFilesAndPush(
	ctx context.Context,
	repoPath string,
	filePaths []string,
	branch string,
	env []string,
	author *rpc.Identity,
	authorDate time.Time,
	committer *rpc.Identity,
	committerDate time.Time,
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
	err = s.adapter.Commit(ctx, repoPath, types.CommitChangesOptions{
		Committer: types.Signature{
			Identity: types.Identity{
				Name:  committer.Name,
				Email: committer.Email,
			},
			When: committerDate,
		},
		Author: types.Signature{
			Identity: types.Identity{
				Name:  author.Name,
				Email: author.Email,
			},
			When: authorDate,
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
		Env:     env,
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
	_, err = s.store.Save(fullPath, &fileData)
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
