// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"io"

	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/rpc"
)

type GetBlobParams struct {
	ReadParams
	SHA       string
	SizeLimit int64
}

type GetBlobOutput struct {
	SHA string
	// Size is the actual size of the blob.
	Size int64
	// ContentSize is the total number of bytes returned by the Content Reader.
	ContentSize int64
	// Content contains the (partial) content of the blob.
	Content io.Reader
}

func (c *Client) GetBlob(ctx context.Context, params *GetBlobParams) (*GetBlobOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	stream, err := c.repoService.GetBlob(ctx, &rpc.GetBlobRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		Sha:       params.SHA,
		SizeLimit: params.SizeLimit,
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to start blob stream")
	}

	msg, err := stream.Recv()
	if err != nil {
		return nil, processRPCErrorf(err, "failed to read blob header from stream")
	}

	header := msg.GetHeader()
	if header == nil {
		return nil, Errorf(StatusInternal, "expected to receive header from server")
	}

	// setup contentReader that reads content from grpc stream
	contentReader := streamio.NewReader(func() ([]byte, error) {
		resp, rErr := stream.Recv()
		return resp.GetContent(), rErr
	})

	return &GetBlobOutput{
		SHA:         header.GetSha(),
		Size:        header.GetSize(),
		ContentSize: header.GetContentSize(),
		Content:     contentReader,
	}, nil
}
