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

type UpdatePipelineOutput struct {
	Status string       `json:"status"`
	Data   PipelineData `json:"data"`
}

type UpdatePipelineInput struct {
	Prompt   string `json:"prompt"`
	RepoRef  string `json:"repo_ref"`
	Pipeline string `json:"pipeline"`
}

func (c *Controller) UpdatePipeline(
	ctx context.Context,
	in *UpdatePipelineInput,
) (*UpdatePipelineOutput, error) {
	generateRequest := &types.PipelineUpdateRequest{
		Prompt:   in.Prompt,
		RepoRef:  in.RepoRef,
		Pipeline: in.Pipeline,
	}

	// do permission check on repo here?
	repo, err := c.repoStore.FindByRef(ctx, in.RepoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}

	output, err := c.intelligenceService.Update(ctx, generateRequest, repo)
	if err != nil {
		return nil, fmt.Errorf("update pipeline: %w", err)
	}
	return &UpdatePipelineOutput{
		Status: "SUCCESS",
		Data: PipelineData{
			YamlPipeline: output.YAML,
		},
	}, nil
}
