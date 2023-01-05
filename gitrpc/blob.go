// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc/rpc"
)

type GetBlobParams struct {
	ReadParams
	SHA       string
	SizeLimit int64
}

type GetBlobOutput struct {
	Blob Blob
}

type Blob struct {
	SHA  string
	Size int64
	// Content contains the data of the blob
	// NOTE: can be only partial data - compare len(.content) with .size
	Content []byte
}

func (c *Client) GetBlob(ctx context.Context, params *GetBlobParams) (*GetBlobOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	resp, err := c.repoService.GetBlob(ctx, &rpc.GetBlobRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		Sha:       params.SHA,
		SizeLimit: params.SizeLimit,
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to get blob from server")
	}

	blob := resp.GetBlob()
	if blob == nil {
		return nil, fmt.Errorf("rpc blob is nil")
	}

	return &GetBlobOutput{
		Blob: Blob{
			SHA:     blob.GetSha(),
			Size:    blob.GetSize(),
			Content: blob.GetContent(),
		},
	}, nil
}
