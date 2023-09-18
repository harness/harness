// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
)

type GeneratePipelineParams struct {
	ReadParams
}

type GeneratePipelinesOutput struct {
	PipelineYAML []byte
}

func (c *Client) GeneratePipeline(ctx context.Context,
	params *GeneratePipelineParams,
) (GeneratePipelinesOutput, error) {
	response, err := c.repoService.GeneratePipeline(ctx, &rpc.GeneratePipelineRequest{
		Base: mapToRPCReadRequest(params.ReadParams),
	})
	if err != nil {
		return GeneratePipelinesOutput{}, processRPCErrorf(err, "failed to generate pipeline")
	}

	return GeneratePipelinesOutput{
		PipelineYAML: response.PipelineYaml,
	}, nil
}
