// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc/rpc"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/rs/zerolog/log"
)

const (
	// repoGitUIDLength is the length of the generated repo uid.
	repoGitUIDLength = 42

	// repoGitUIDAlphabet is the alphabet used for generating repo uids
	// NOTE: keep it lowercase and alphanumerical to avoid issues with case insensitive filesystems.
	repoGitUIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
)

type CreateRepositoryParams struct {
	// Create operation is different from all (from user side), as UID doesn't exist yet.
	// Only take actor as input and create WriteParams manually
	Actor         Identity
	DefaultBranch string
	Files         []File
}

type CreateRepositoryOutput struct {
	UID string
}

func (c *Client) CreateRepository(ctx context.Context,
	params *CreateRepositoryParams) (*CreateRepositoryOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	log := log.Ctx(ctx)

	uid, err := newRepositoryUID()
	if err != nil {
		return nil, fmt.Errorf("failed to create new uid: %w", err)
	}
	log.Info().
		Msgf("Create new git repository with uid '%s' and default branch '%s'", uid, params.DefaultBranch)

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	stream, err := c.repoService.CreateRepository(ctx)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("Send header")

	writeParams := WriteParams{
		RepoUID: uid,
		Actor:   params.Actor,
		EnvVars: map[string]string{}, // (no githook triggered for repo creation)
	}

	req := &rpc.CreateRepositoryRequest{
		Data: &rpc.CreateRepositoryRequest_Header{
			Header: &rpc.CreateRepositoryRequestHeader{
				Base:          mapToRPCWriteRequest(writeParams),
				DefaultBranch: params.DefaultBranch,
			},
		},
	}
	if err = stream.Send(req); err != nil {
		return nil, err
	}

	for _, file := range params.Files {
		log.Info().Msgf("Send file %s", file.Path)

		err = uploadFile(ctx, file, FileTransferChunkSize, func(fs *rpc.FileUpload) error {
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
		return nil, processRPCErrorf(err, "failed to create repo on server (uid: '%s')", uid)
	}

	log.Info().Msgf("completed git repo setup.")

	return &CreateRepositoryOutput{UID: uid}, nil
}

func newRepositoryUID() (string, error) {
	return gonanoid.Generate(repoGitUIDAlphabet, repoGitUIDLength)
}
