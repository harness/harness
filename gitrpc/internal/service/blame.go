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

	commit, err := mapGitCommit(&part.Commit)
	if err != nil {
		return fmt.Errorf("failed to map git commit: %w", err)
	}

	lines := make([][]byte, len(part.Lines))
	for i, line := range part.Lines {
		lines[i] = []byte(line)
	}

	pack := &rpc.BlamePart{
		Commit: commit,
		Lines:  lines,
	}

	if err = stream.Send(pack); err != nil {
		return fmt.Errorf("failed to send blame part: %w", err)
	}

	return nil
}
