// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/rpc"
)

type RawDiffParams struct {
	ReadParams
	BaseRef   string
	HeadRef   string
	MergeBase bool
}

func (c *Client) RawDiff(ctx context.Context, params *RawDiffParams, out io.Writer) error {
	diff, err := c.diffService.RawDiff(ctx, &rpc.RawDiffRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		BaseRef:   params.BaseRef,
		HeadRef:   params.HeadRef,
		MergeBase: params.MergeBase,
	})
	if err != nil {
		return err
	}

	reader := streamio.NewReader(func() ([]byte, error) {
		var resp *rpc.RawDiffResponse
		resp, err = diff.Recv()
		return resp.GetData(), err
	})

	if _, err = io.Copy(out, reader); err != nil {
		return fmt.Errorf("copy rpc data: %w", err)
	}

	return nil
}
