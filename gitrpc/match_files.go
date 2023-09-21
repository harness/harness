// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
)

type FileContent struct {
	Path    string
	Content []byte
}

type MatchFilesParams struct {
	ReadParams
	Ref     string
	DirPath string
	Pattern string
	MaxSize int
}

type MatchFilesOutput struct {
	Files []FileContent
}

func (c *Client) MatchFiles(ctx context.Context,
	params *MatchFilesParams,
) (*MatchFilesOutput, error) {
	resp, err := c.repoService.MatchFiles(ctx, &rpc.MatchFilesRequest{
		Base:    mapToRPCReadRequest(params.ReadParams),
		Ref:     params.Ref,
		DirPath: params.DirPath,
		Pattern: params.Pattern,
		MaxSize: int32(params.MaxSize),
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to match files")
	}

	files := make([]FileContent, len(resp.Files))
	for i, f := range resp.Files {
		files[i] = FileContent{
			Path:    f.Path,
			Content: f.Content,
		}
	}

	return &MatchFilesOutput{
		Files: files,
	}, nil
}
