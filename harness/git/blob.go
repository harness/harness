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

package git

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"
)

type GetBlobParams struct {
	ReadParams
	SHA       string
	SizeLimit int64
}

type GetBlobOutput struct {
	SHA sha.SHA
	// Size is the actual size of the blob.
	Size int64
	// ContentSize is the total number of bytes returned by the Content Reader.
	ContentSize int64
	// Content contains the (partial) content of the blob.
	Content io.ReadCloser
}

func (s *Service) GetBlob(ctx context.Context, params *GetBlobParams) (*GetBlobOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	// TODO: do we need to validate request for nil?
	reader, err := api.GetBlob(
		ctx,
		repoPath,
		params.AlternateObjectDirs,
		sha.Must(params.SHA),
		params.SizeLimit,
	)
	if err != nil {
		return nil, err
	}

	return &GetBlobOutput{
		SHA:         reader.SHA,
		Size:        reader.Size,
		ContentSize: reader.ContentSize,
		Content:     reader.Content,
	}, nil
}

func (s *Service) FindLFSPointers(
	ctx context.Context,
	params *FindLFSPointersParams,
) (*FindLFSPointersOutput, error) {
	if params.RepoUID == "" {
		return nil, api.ErrRepositoryPathEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	var objects []parser.BatchCheckObject
	for _, gitObjDir := range params.AlternateObjectDirs {
		objs, err := s.listGitObjDir(ctx, repoPath, gitObjDir)
		if err != nil {
			return nil, err
		}
		objects = append(objects, objs...)
	}

	var candidateObjects []parser.BatchCheckObject
	for _, obj := range objects {
		if obj.Type == string(TreeNodeTypeBlob) && obj.Size <= parser.LfsPointerMaxSize {
			candidateObjects = append(candidateObjects, obj)
		}
	}

	var lfsInfos []LFSInfo
	if len(candidateObjects) == 0 {
		return &FindLFSPointersOutput{LFSInfos: lfsInfos}, nil
	}

	// check the short-listed objects for lfs-pointers content
	stdIn, stdOut, cancel := api.CatFileBatch(ctx, repoPath, params.AlternateObjectDirs)
	defer cancel()

	for _, obj := range candidateObjects {
		line := obj.SHA.String() + "\n"

		_, err := stdIn.Write([]byte(line))
		if err != nil {
			return nil, fmt.Errorf("failed to write blob sha to git stdin: %w", err)
		}

		// first line is always the object type, sha, and size
		_, err = stdOut.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read the git cat-file output: %w", err)
		}

		content, err := io.ReadAll(io.LimitReader(stdOut, obj.Size))
		if err != nil {
			return nil, fmt.Errorf("failed to read the git cat-file output: %w", err)
		}

		oid, err := parser.GetLFSObjectID(content)
		if err != nil && !errors.Is(err, parser.ErrInvalidLFSPointer) {
			return nil, fmt.Errorf("failed to scan git cat-file output for %s: %w", obj.SHA, err)
		}
		if err == nil {
			lfsInfos = append(lfsInfos, LFSInfo{ObjID: oid, SHA: obj.SHA})
		}

		// skip the trailing new line
		_, err = stdOut.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read trailing newline after object: %w", err)
		}
	}

	return &FindLFSPointersOutput{LFSInfos: lfsInfos}, nil
}
