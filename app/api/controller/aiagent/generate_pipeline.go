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

package aiagent

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
)

type GeneratePipelineInput struct {
	Prompt  string `json:"prompt"`
	RepoRef string `json:"repo_ref"`
}

type PipelineData struct {
	YamlPipeline string `json:"yaml_pipeline"`
}

type GeneratePipelineOutput struct {
	Status string       `json:"status"`
	Data   PipelineData `json:"data"`
}

func (c *Controller) GeneratePipeline(
	ctx context.Context,
	in *GeneratePipelineInput,
) (*GeneratePipelineOutput, error) {
	generateRequest := &types.PipelineGenerateRequest{
		Prompt:  in.Prompt,
		RepoRef: in.RepoRef,
	}

	// do permission check on repo here?
	repo, err := c.repoStore.FindByRef(ctx, in.RepoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}

	output, err := c.intelligenceService.Generate(ctx, generateRequest, repo)
	if err != nil {
		return nil, fmt.Errorf("generate pipeline: %w", err)
	}
	return &GeneratePipelineOutput{
		Status: "SUCCESS",
		Data: PipelineData{
			YamlPipeline: output.YAML,
		},
	}, nil
}
