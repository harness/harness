// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"
)

type BlameService struct {
	rpc.UnimplementedBlameServiceServer
	adapter   GitAdapter
	reposRoot string
}

var _ rpc.BlameServiceServer = (*BlameService)(nil)

func NewBlameService(adapter GitAdapter, reposRoot string) *BlameService {
	return &BlameService{
		adapter:   adapter,
		reposRoot: reposRoot,
	}
}

func (s BlameService) Blame(request *rpc.BlameRequest, stream rpc.BlameService_BlameServer) error {
	ctx := stream.Context()

	repoPath := getFullPathForRepo(s.reposRoot, request.Base.GetRepoUid())

	reader := s.adapter.Blame(ctx,
		repoPath, request.GitRef, request.Path,
		int(request.Range.From), int(request.Range.To))

	for {
		part, errRead := reader.NextPart()

		errStream := streamBlamePart(part, stream)
		if errStream != nil {
			return errStream
		}

		if errRead != nil {
			if errors.Is(errRead, io.EOF) {
				return nil
			}
			return errRead
		}
	}
}

func streamBlamePart(
	part *types.BlamePart, stream rpc.BlameService_BlameServer,
) error {
	if part == nil {
		return nil
	}

	commit, errMap := mapGitCommit(&part.Commit)
	if errMap != nil {
		return fmt.Errorf("failed to map git commit: %w", errMap)
	}

	lines := make([][]byte, len(part.Lines))
	for i, line := range part.Lines {
		lines[i] = []byte(line)
	}

	pack := &rpc.BlamePart{
		Commit: commit,
		Lines:  lines,
	}

	if errStream := stream.Send(pack); errStream != nil {
		return errStream
	}

	return nil
}
