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
	"strings"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"
)

const (
	maxOversizeFiles       = 10
	maxCommitterMismatches = 10
)

type FindOversizeFilesParams struct {
	// TODO: remove. Kept for backward compatibility
	RepoUID       string
	GitObjectDirs []string

	SizeLimit int64
}

type FileInfo struct {
	SHA  sha.SHA
	Size int64
}

type FindOversizeFilesOutput struct {
	FileInfos []FileInfo
	Total     int64
}

type FindCommitterMismatchParams struct {
	PrincipalEmail string
}

type CommitInfo struct {
	SHA       sha.SHA
	Committer string
}

type FindCommitterMismatchOutput struct {
	CommitInfos []CommitInfo
	Total       int64
}

type ProcessPreReceiveObjectsParams struct {
	ReadParams
	FindOversizeFilesParams     *FindOversizeFilesParams
	FindCommitterMismatchParams *FindCommitterMismatchParams
}

type ProcessPreReceiveObjectsOutput struct {
	FindOversizeFilesOutput     *FindOversizeFilesOutput
	FindCommitterMismatchOutput *FindCommitterMismatchOutput
}

func (s *Service) ProcessPreReceiveObjects(
	ctx context.Context,
	params ProcessPreReceiveObjectsParams,
) (ProcessPreReceiveObjectsOutput, error) {
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	var objects []parser.BatchCheckObject
	for _, gitObjDir := range params.AlternateObjectDirs {
		objs, err := s.listGitObjDir(ctx, repoPath, gitObjDir)
		if err != nil {
			return ProcessPreReceiveObjectsOutput{}, err
		}
		objects = append(objects, objs...)
	}

	var output ProcessPreReceiveObjectsOutput

	if params.FindOversizeFilesParams != nil {
		output.FindOversizeFilesOutput = findOversizeFiles(
			objects, params.FindOversizeFilesParams,
		)
	}

	if params.FindCommitterMismatchParams != nil {
		out, err := findCommitterMismatch(
			ctx,
			objects,
			repoPath,
			params.ReadParams.AlternateObjectDirs,
			params.FindCommitterMismatchParams,
		)
		if err != nil {
			return ProcessPreReceiveObjectsOutput{}, err
		}
		output.FindCommitterMismatchOutput = out
	}

	return output, nil
}

func findOversizeFiles(
	objects []parser.BatchCheckObject,
	findOversizeFilesParams *FindOversizeFilesParams,
) *FindOversizeFilesOutput {
	var fileInfos []FileInfo

	var total int64 // limit the total num of objects returned
	for _, obj := range objects {
		if obj.Type == string(TreeNodeTypeBlob) && obj.Size > findOversizeFilesParams.SizeLimit {
			if total < maxOversizeFiles {
				fileInfos = append(fileInfos, FileInfo{
					SHA:  obj.SHA,
					Size: obj.Size,
				})
			}
			total++
		}
	}

	return &FindOversizeFilesOutput{
		FileInfos: fileInfos,
		Total:     total,
	}
}

func findCommitterMismatch(
	ctx context.Context,
	objects []parser.BatchCheckObject,
	repoPath string,
	alternateObjectDirs []string,
	findCommitterEmailsMismatchParams *FindCommitterMismatchParams,
) (*FindCommitterMismatchOutput, error) {
	var commitSHAs []string
	for _, obj := range objects {
		if obj.Type == string(TreeNodeTypeCommit) {
			commitSHAs = append(commitSHAs, obj.SHA.String())
		}
	}

	writer, reader, cancel := api.CatFileBatch(ctx, repoPath, alternateObjectDirs)
	defer cancel()
	defer writer.Close()

	var total int64
	var commitInfos []CommitInfo
	for _, commitSHA := range commitSHAs {
		_, writeErr := writer.Write([]byte(commitSHA + "\n"))
		if writeErr != nil {
			return nil, fmt.Errorf("failed to write to cat-file batch: %w", writeErr)
		}

		output, err := api.ReadBatchHeaderLine(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read cat-file batch header: %w", err)
		}

		limitedReader := io.LimitReader(reader, output.Size+1) // plus eol

		data, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read: %w", err)
		}
		text := strings.Split(string(data), "\n")

		for _, line := range text {
			if !strings.HasPrefix(line, "committer ") {
				continue
			}

			committerEmail := line[strings.Index(line, "<")+1 : strings.Index(line, ">")]
			if !strings.EqualFold(committerEmail, findCommitterEmailsMismatchParams.PrincipalEmail) {
				if total < maxCommitterMismatches {
					sha, err := sha.New(commitSHA)
					if err != nil {
						return nil, fmt.Errorf("failed to create new sha: %w", err)
					}
					commitInfos = append(commitInfos, CommitInfo{
						SHA:       sha,
						Committer: committerEmail,
					})
				}
				total++
			}

			break
		}
	}

	return &FindCommitterMismatchOutput{
		CommitInfos: commitInfos,
		Total:       total,
	}, nil
}
