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
