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
	"io"

	"github.com/harness/gitness/git/api"
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
