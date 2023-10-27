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

package gitrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc/hash"
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
	// Only take actor and envars as input and create WriteParams manually
	Actor   Identity
	EnvVars map[string]string

	DefaultBranch string
	Files         []File

	// Committer overwrites the git committer used for committing the files
	// (optional, default: actor)
	Committer *Identity
	// CommitterDate overwrites the git committer date used for committing the files
	// (optional, default: current time on server)
	CommitterDate *time.Time
	// Author overwrites the git author used for committing the files
	// (optional, default: committer)
	Author *Identity
	// AuthorDate overwrites the git author date used for committing the files
	// (optional, default: committer date)
	AuthorDate *time.Time
}

type CreateRepositoryOutput struct {
	UID string
}

type DeleteRepositoryParams struct {
	WriteParams
}

type SyncRepositoryParams struct {
	WriteParams
	Source            string
	CreateIfNotExists bool

	// RefSpecs [OPTIONAL] allows to override the refspecs that are being synced from the remote repository.
	// By default all references present on the remote repository will be fetched (including scm internal ones).
	RefSpecs []string
}

type SyncRepositoryOutput struct {
	DefaultBranch string
}

type HashRepositoryParams struct {
	ReadParams
	HashType        hash.Type
	AggregationType hash.AggregationType
}

type HashRepositoryOutput struct {
	Hash []byte
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
		EnvVars: params.EnvVars,
	}

	req := &rpc.CreateRepositoryRequest{
		Data: &rpc.CreateRepositoryRequest_Header{
			Header: &rpc.CreateRepositoryRequestHeader{
				Base:          mapToRPCWriteRequest(writeParams),
				DefaultBranch: params.DefaultBranch,
				Author:        mapToRPCIdentityOptional(params.Author),
				AuthorDate:    mapToRPCTimeOptional(params.AuthorDate),
				Committer:     mapToRPCIdentityOptional(params.Committer),
				CommitterDate: mapToRPCTimeOptional(params.CommitterDate),
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

func (c *Client) DeleteRepository(ctx context.Context, params *DeleteRepositoryParams) error {
	if params == nil {
		return ErrNoParamsProvided
	}
	_, err := c.repoService.DeleteRepository(ctx, &rpc.DeleteRepositoryRequest{
		Base: mapToRPCWriteRequest(params.WriteParams),
	})
	if err != nil {
		return processRPCErrorf(err, "failed to delete repository on server")
	}
	return nil
}

func (c *Client) SyncRepository(ctx context.Context, params *SyncRepositoryParams) (*SyncRepositoryOutput, error) {
	result, err := c.repoService.SyncRepository(ctx, &rpc.SyncRepositoryRequest{
		Base:              mapToRPCWriteRequest(params.WriteParams),
		Source:            params.Source,
		CreateIfNotExists: params.CreateIfNotExists,
		RefSpecs:          params.RefSpecs,
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to sync repository on server to match provided source")
	}

	return &SyncRepositoryOutput{
		DefaultBranch: result.DefaultBranch,
	}, nil
}

func (c *Client) HashRepository(ctx context.Context, params *HashRepositoryParams) (*HashRepositoryOutput, error) {
	hashType, err := mapToRPCHashType(params.HashType)
	if err != nil {
		return nil, fmt.Errorf("failed to map hash type: %w", err)
	}
	aggregationType, err := mapToRPCHashAggregationType(params.AggregationType)
	if err != nil {
		return nil, fmt.Errorf("failed to map aggregation type: %w", err)
	}

	resp, err := c.repoService.HashRepository(ctx, &rpc.HashRepositoryRequest{
		Base:            mapToRPCReadRequest(params.ReadParams),
		HashType:        hashType,
		AggregationType: aggregationType,
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to hash repository on server")
	}

	return &HashRepositoryOutput{
		Hash: resp.GetHash(),
	}, nil
}
