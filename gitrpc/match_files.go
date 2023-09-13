// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
